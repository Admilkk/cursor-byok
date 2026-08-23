use sqlx::Row;

use crate::{
    model::{CursorRunTraceArtifact, CursorRunTraceSummary},
    Result,
};

use super::{now_ms, BlobId, Store};

impl Store {
    pub async fn start_cursor_trace_if_detailed(
        &self,
        request_id: &str,
        conversation_id: Option<&str>,
        route: &str,
        model_id: Option<&str>,
    ) -> Result<bool> {
        if self.cursor_trace_exists(request_id).await? {
            return Ok(true);
        }
        if !self.detailed_logging().await? {
            return Ok(false);
        }
        sqlx::query(
            "INSERT OR IGNORE INTO cursor_run_traces(
                request_id, conversation_id, route, model_id, status, received_at_ms
             ) VALUES (?, ?, ?, ?, 'running', ?)",
        )
        .bind(request_id)
        .bind(conversation_id)
        .bind(route)
        .bind(model_id)
        .bind(now_ms())
        .execute(&self.pool)
        .await?;
        Ok(true)
    }

    pub async fn cursor_trace_exists(&self, request_id: &str) -> Result<bool> {
        Ok(sqlx::query_scalar::<_, bool>(
            "SELECT EXISTS(SELECT 1 FROM cursor_run_traces WHERE request_id = ?)",
        )
        .bind(request_id)
        .fetch_one(&self.pool)
        .await?)
    }

    pub async fn append_cursor_trace_artifact(
        &self,
        request_id: &str,
        artifact_type: &str,
        source: &str,
        data: &[u8],
        metadata: &serde_json::Value,
    ) -> Result<()> {
        let blob_id = self.put_blob(data, &[]).await?;
        self.link_cursor_trace_artifact(request_id, artifact_type, source, &blob_id, metadata)
            .await
    }

    pub async fn link_cursor_trace_artifact(
        &self,
        request_id: &str,
        artifact_type: &str,
        source: &str,
        blob_id: &BlobId,
        metadata: &serde_json::Value,
    ) -> Result<()> {
        let mut tx = self.pool.begin_with("BEGIN IMMEDIATE").await?;
        let next: i64 = sqlx::query_scalar(
            "SELECT COALESCE(MAX(seq), -1) + 1
             FROM cursor_run_trace_artifacts WHERE request_id = ?",
        )
        .bind(request_id)
        .fetch_one(&mut *tx)
        .await?;
        sqlx::query(
            "INSERT INTO cursor_run_trace_artifacts(
                request_id, seq, artifact_type, source, blob_id, metadata_json, created_at_ms
             ) VALUES (?, ?, ?, ?, ?, ?, ?)",
        )
        .bind(request_id)
        .bind(next)
        .bind(artifact_type)
        .bind(source)
        .bind(blob_id.as_bytes().as_slice())
        .bind(serde_json::to_string(metadata)?)
        .bind(now_ms())
        .execute(&mut *tx)
        .await?;
        tx.commit().await?;
        Ok(())
    }

    pub async fn add_cursor_trace_request_bytes(
        &self,
        request_id: &str,
        bytes: usize,
    ) -> Result<()> {
        sqlx::query(
            "UPDATE cursor_run_traces
             SET request_bytes = request_bytes + ? WHERE request_id = ?",
        )
        .bind(as_i64(bytes))
        .bind(request_id)
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    pub async fn start_cursor_trace_response(&self, request_id: &str, status: u16) -> Result<()> {
        let now = now_ms();
        sqlx::query(
            "UPDATE cursor_run_traces
             SET status = 'running', http_status = ?,
                 first_response_at_ms = COALESCE(first_response_at_ms, ?),
                 finished_at_ms = NULL, error_message = NULL
             WHERE request_id = ?",
        )
        .bind(status as i64)
        .bind(now)
        .bind(request_id)
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    pub async fn add_cursor_trace_response_chunk(
        &self,
        request_id: &str,
        source: &str,
        data: &[u8],
    ) -> Result<()> {
        self.append_cursor_trace_artifact(
            request_id,
            "run_sse_chunk",
            source,
            data,
            &serde_json::json!({"byte_count": data.len()}),
        )
        .await?;
        sqlx::query(
            "UPDATE cursor_run_traces
             SET response_bytes = response_bytes + ?,
                 response_event_count = response_event_count + 1,
                 first_response_at_ms = COALESCE(first_response_at_ms, ?)
             WHERE request_id = ?",
        )
        .bind(as_i64(data.len()))
        .bind(now_ms())
        .bind(request_id)
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    pub async fn finish_cursor_trace(&self, request_id: &str, error: Option<&str>) -> Result<()> {
        sqlx::query(
            "UPDATE cursor_run_traces
             SET status = ?, finished_at_ms = ?, error_message = ?
             WHERE request_id = ?",
        )
        .bind(if error.is_some() {
            "error"
        } else {
            "completed"
        })
        .bind(now_ms())
        .bind(error)
        .bind(request_id)
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    pub async fn cursor_trace(&self, request_id: &str) -> Result<Option<CursorRunTraceSummary>> {
        sqlx::query("SELECT * FROM cursor_run_traces WHERE request_id = ?")
            .bind(request_id)
            .fetch_optional(&self.pool)
            .await?
            .map(trace_from_row)
            .transpose()
    }

    pub async fn official_cursor_traces(&self, limit: i64) -> Result<Vec<CursorRunTraceSummary>> {
        let rows = sqlx::query(
            "SELECT * FROM cursor_run_traces
             WHERE route = 'cursor_official'
             ORDER BY received_at_ms DESC LIMIT ?",
        )
        .bind(limit.clamp(1, 500))
        .fetch_all(&self.pool)
        .await?;
        rows.into_iter().map(trace_from_row).collect()
    }

    pub async fn cursor_trace_artifacts(
        &self,
        request_id: &str,
    ) -> Result<Vec<CursorRunTraceArtifact>> {
        let rows = sqlx::query(
            "SELECT a.seq, a.artifact_type, a.source, a.metadata_json,
                    a.created_at_ms, b.data
             FROM cursor_run_trace_artifacts a
             JOIN blobs b ON b.blob_id = a.blob_id
             WHERE a.request_id = ? ORDER BY a.seq",
        )
        .bind(request_id)
        .fetch_all(&self.pool)
        .await?;
        rows.into_iter()
            .map(|row| {
                Ok(CursorRunTraceArtifact {
                    seq: row.try_get("seq")?,
                    artifact_type: row.try_get("artifact_type")?,
                    source: row.try_get("source")?,
                    metadata: serde_json::from_str(row.try_get("metadata_json")?)?,
                    created_at_ms: row.try_get("created_at_ms")?,
                    data: row.try_get("data")?,
                })
            })
            .collect()
    }
}

fn trace_from_row(row: sqlx::sqlite::SqliteRow) -> Result<CursorRunTraceSummary> {
    Ok(CursorRunTraceSummary {
        request_id: row.try_get("request_id")?,
        conversation_id: row.try_get("conversation_id")?,
        route: row.try_get("route")?,
        model_id: row.try_get("model_id")?,
        status: row.try_get("status")?,
        request_bytes: row.try_get("request_bytes")?,
        response_bytes: row.try_get("response_bytes")?,
        response_event_count: row.try_get("response_event_count")?,
        http_status: row.try_get("http_status")?,
        received_at_ms: row.try_get("received_at_ms")?,
        first_response_at_ms: row.try_get("first_response_at_ms")?,
        finished_at_ms: row.try_get("finished_at_ms")?,
        error_message: row.try_get("error_message")?,
    })
}

fn as_i64(value: usize) -> i64 {
    value.min(i64::MAX as usize) as i64
}
