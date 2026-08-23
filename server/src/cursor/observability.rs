use crate::{store::BlobId, store::Store};

#[derive(Clone)]
pub struct CursorTraceRecorder {
    store: Store,
    request_id: String,
}

impl CursorTraceRecorder {
    pub async fn begin(
        store: Store,
        request_id: &str,
        conversation_id: Option<&str>,
        route: &str,
        model_id: Option<&str>,
    ) -> Option<Self> {
        match store
            .start_cursor_trace_if_detailed(request_id, conversation_id, route, model_id)
            .await
        {
            Ok(true) => Some(Self {
                store,
                request_id: request_id.into(),
            }),
            Ok(false) => None,
            Err(error) => {
                tracing::warn!(request_id, %error, "failed to start Cursor trace");
                None
            }
        }
    }

    pub async fn resume(store: Store, request_id: &str) -> Option<Self> {
        match store.cursor_trace_exists(request_id).await {
            Ok(true) => Some(Self {
                store,
                request_id: request_id.into(),
            }),
            Ok(false) => None,
            Err(error) => {
                tracing::warn!(request_id, %error, "failed to resume Cursor trace");
                None
            }
        }
    }

    pub fn request_id(&self) -> &str {
        &self.request_id
    }

    pub async fn request(&self, artifact_type: &str, data: &[u8], metadata: serde_json::Value) {
        if let Err(error) = self
            .store
            .append_cursor_trace_artifact(
                &self.request_id,
                artifact_type,
                "cursor_client",
                data,
                &metadata,
            )
            .await
        {
            tracing::warn!(request_id = self.request_id, %error, "failed to record Cursor request artifact");
            return;
        }
        if let Err(error) = self
            .store
            .add_cursor_trace_request_bytes(&self.request_id, data.len())
            .await
        {
            tracing::warn!(request_id = self.request_id, %error, "failed to update Cursor request trace size");
        }
    }

    pub async fn artifact(
        &self,
        artifact_type: &str,
        source: &str,
        data: &[u8],
        metadata: serde_json::Value,
    ) {
        if let Err(error) = self
            .store
            .append_cursor_trace_artifact(&self.request_id, artifact_type, source, data, &metadata)
            .await
        {
            tracing::warn!(request_id = self.request_id, %error, artifact_type, "failed to record Cursor trace artifact");
        }
    }

    pub async fn linked_blob(
        &self,
        artifact_type: &str,
        source: &str,
        blob_id: &BlobId,
        metadata: serde_json::Value,
    ) {
        if let Err(error) = self
            .store
            .link_cursor_trace_artifact(&self.request_id, artifact_type, source, blob_id, &metadata)
            .await
        {
            tracing::warn!(request_id = self.request_id, %error, artifact_type, "failed to link Cursor trace Blob");
        }
    }

    pub async fn response_started(&self, status: u16) {
        if let Err(error) = self
            .store
            .start_cursor_trace_response(&self.request_id, status)
            .await
        {
            tracing::warn!(request_id = self.request_id, %error, "failed to start Cursor response trace");
        }
    }

    pub async fn response_chunk(&self, source: &str, data: &[u8]) {
        if let Err(error) = self
            .store
            .add_cursor_trace_response_chunk(&self.request_id, source, data)
            .await
        {
            tracing::warn!(request_id = self.request_id, %error, "failed to record Cursor response chunk");
        }
    }

    pub async fn finish(&self, error: Option<&str>) {
        if let Err(store_error) = self
            .store
            .finish_cursor_trace(&self.request_id, error)
            .await
        {
            tracing::warn!(request_id = self.request_id, %store_error, "failed to finish Cursor trace");
        }
    }
}
