//! Storage accounting and cleanup for disposable observability data.

use serde::Serialize;

use crate::Result;

use super::Store;

#[derive(Clone, Copy, Debug, Default, Serialize)]
pub struct StatisticsStorage {
    pub bytes: i64,
    pub call_count: i64,
    pub trace_count: i64,
}

impl Store {
    pub async fn statistics_storage(&self) -> Result<StatisticsStorage> {
        let (bytes, call_count, trace_count) = sqlx::query_as::<_, (i64, i64, i64)>(
            r#"
            SELECT
                COALESCE((
                    SELECT SUM(
                        LENGTH(call_id) + LENGTH(run_id) + LENGTH(conversation_id) +
                        LENGTH(provider_type) + LENGTH(provider_url) + LENGTH(request_type) +
                        LENGTH(request_url) + LENGTH(model_id) + LENGTH(display_name) +
                        LENGTH(status) + COALESCE(LENGTH(finish_reason), 0) +
                        COALESCE(LENGTH(usage_json), 0) + COALESCE(LENGTH(error_kind), 0) +
                        COALESCE(LENGTH(error_message), 0) + 256
                    ) FROM llm_calls
                ), 0) +
                COALESCE((SELECT SUM(LENGTH(headers_json) + LENGTH(body_json) + 24) FROM llm_call_requests), 0) +
                COALESCE((SELECT SUM(LENGTH(data) + 24) FROM llm_call_response_chunks), 0) +
                COALESCE((
                    SELECT SUM(
                        LENGTH(request_id) + COALESCE(LENGTH(conversation_id), 0) +
                        LENGTH(route) + COALESCE(LENGTH(model_id), 0) + LENGTH(status) +
                        COALESCE(LENGTH(error_message), 0) + 96
                    ) FROM cursor_run_traces
                ), 0) +
                COALESCE((SELECT SUM(LENGTH(artifact_type) + LENGTH(source) + LENGTH(metadata_json) + 48) FROM cursor_run_trace_artifacts), 0) +
                COALESCE((SELECT SUM(LENGTH(data)) FROM blobs WHERE blob_id IN (SELECT blob_id FROM cursor_run_trace_artifacts)), 0),
                (SELECT COUNT(*) FROM llm_calls),
                (SELECT COUNT(*) FROM cursor_run_traces)
            "#,
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(StatisticsStorage {
            bytes,
            call_count,
            trace_count,
        })
    }

    pub async fn clear_statistics_storage(&self) -> Result<StatisticsStorage> {
        let mut transaction = self.pool.begin().await?;
        sqlx::query("DELETE FROM llm_calls")
            .execute(&mut *transaction)
            .await?;
        sqlx::query("DELETE FROM cursor_run_traces")
            .execute(&mut *transaction)
            .await?;
        transaction.commit().await?;
        self.statistics_storage().await
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn clears_observability_without_removing_configuration() {
        let store = Store::connect("sqlite::memory:").await.unwrap();
        sqlx::query("INSERT INTO provider_endpoints(name, provider_type, base_url, api_key, created_at_ms, updated_at_ms) VALUES ('Example', 'openai-chat', 'https://example.com', 'secret', 1, 1)")
            .execute(store.pool()).await.unwrap();
        sqlx::query("INSERT INTO llm_calls(call_id, run_id, conversation_id, provider_call_index, provider_type, provider_url, request_type, request_url, model_id, display_name, status, created_at_ms, message_count, tool_count, detailed) VALUES ('call-1', 'run-1', 'conversation-1', 0, 'openai-chat', 'https://example.com', 'openai-chat', 'https://example.com/v1/chat/completions', 'model', 'Model', 'completed', 1, 1, 0, 0)")
            .execute(store.pool()).await.unwrap();

        assert!(store.statistics_storage().await.unwrap().bytes > 0);
        let cleared = store.clear_statistics_storage().await.unwrap();
        assert_eq!(cleared.bytes, 0);
        assert_eq!(cleared.call_count, 0);
        let provider_count: i64 = sqlx::query_scalar("SELECT COUNT(*) FROM provider_endpoints")
            .fetch_one(store.pool())
            .await
            .unwrap();
        assert_eq!(provider_count, 1);

        store
            .record_llm_request(
                "call-1",
                &serde_json::json!({}),
                &serde_json::json!({"model": "model"}),
                true,
            )
            .await
            .unwrap();
        store
            .record_llm_chunk("call-1", 0, 1, b"data", true)
            .await
            .unwrap();
    }
}
