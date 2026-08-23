use axum::{
    body::Body,
    http::{header, HeaderValue, Response, StatusCode},
};
use bytes::Bytes;
use tokio::sync::mpsc;
use tokio_stream::StreamExt;

use crate::{
    cursor::{observability::CursorTraceRecorder, CursorSessionRegistry},
    Result,
};

pub async fn stream(registry: &CursorSessionRegistry, request_id: &str) -> Result<Response<Body>> {
    let handle = registry.get_or_create(request_id).await?;
    let mut receiver = handle.subscribe();
    let trace = handle.trace().cloned();
    if let Some(trace) = &trace {
        trace.response_started(StatusCode::OK.as_u16()).await;
    }
    let body_stream = async_stream::stream! {
        let mut trace = TraceStreamSink::new(trace, "byok_server");
        while let Some(chunk) = receiver.recv().await {
            trace.chunk(&chunk);
            yield Ok::<Bytes, std::convert::Infallible>(chunk);
        }
        trace.finish(None);
    };
    let mut response = Response::new(Body::from_stream(body_stream));
    *response.status_mut() = StatusCode::OK;
    response.headers_mut().insert(
        header::CONTENT_TYPE,
        HeaderValue::from_static("text/event-stream"),
    );
    response
        .headers_mut()
        .insert(header::CACHE_CONTROL, HeaderValue::from_static("no-cache"));
    response
        .headers_mut()
        .insert("connect-protocol-version", HeaderValue::from_static("1"));
    Ok(response)
}

pub async fn upstream(
    registry: CursorSessionRegistry,
    request_id: String,
    generation: u64,
    response: Response<Body>,
    trace: Option<CursorTraceRecorder>,
) -> Response<Body> {
    let (parts, body) = response.into_parts();
    if let Some(trace) = &trace {
        trace.response_started(parts.status.as_u16()).await;
    }
    let stream = async_stream::stream! {
        let _guard = UpstreamRunGuard {
            registry,
            request_id,
            generation,
        };
        let mut trace = TraceStreamSink::new(trace, "cursor_official");
        let mut body = body.into_data_stream();
        while let Some(chunk) = body.next().await {
            match chunk {
                Ok(chunk) => {
                    trace.chunk(&chunk);
                    yield Ok::<Bytes, axum::Error>(chunk);
                }
                Err(error) => {
                    trace.finish(Some(error.to_string()));
                    yield Err(error);
                    return;
                }
            }
        }
        trace.finish(None);
    };
    Response::from_parts(parts, Body::from_stream(stream))
}

enum TraceStreamEvent {
    Chunk(Bytes),
    Finish(Option<String>),
}

struct TraceStreamSink {
    sender: Option<mpsc::UnboundedSender<TraceStreamEvent>>,
}

impl TraceStreamSink {
    fn new(trace: Option<CursorTraceRecorder>, source: &'static str) -> Self {
        let Some(trace) = trace else {
            return Self { sender: None };
        };
        let (sender, mut receiver) = mpsc::unbounded_channel();
        tokio::spawn(async move {
            while let Some(event) = receiver.recv().await {
                match event {
                    TraceStreamEvent::Chunk(chunk) => {
                        trace.response_chunk(source, &chunk).await;
                    }
                    TraceStreamEvent::Finish(error) => {
                        trace.finish(error.as_deref()).await;
                        return;
                    }
                }
            }
            trace.finish(None).await;
        });
        Self {
            sender: Some(sender),
        }
    }

    fn chunk(&self, chunk: &Bytes) {
        if let Some(sender) = &self.sender {
            let _ = sender.send(TraceStreamEvent::Chunk(chunk.clone()));
        }
    }

    fn finish(&mut self, error: Option<String>) {
        if let Some(sender) = self.sender.take() {
            let _ = sender.send(TraceStreamEvent::Finish(error));
        }
    }
}

impl Drop for TraceStreamSink {
    fn drop(&mut self) {
        self.finish(None);
    }
}

struct UpstreamRunGuard {
    registry: CursorSessionRegistry,
    request_id: String,
    generation: u64,
}

impl Drop for UpstreamRunGuard {
    fn drop(&mut self) {
        self.registry
            .finish_upstream(self.request_id.clone(), self.generation);
    }
}
