use axum::{extract::State, http::StatusCode, Json};
use serde::{Deserialize, Serialize};

use crate::{
    model::{ProviderEndpoint, ProviderEndpointInput, ProviderModel, ProviderModelInput},
    Result,
};

use super::{ControlService, DiscoveredModels};

#[derive(Debug, Deserialize)]
#[serde(tag = "kind", rename_all = "snake_case")]
pub enum ProviderSelection {
    Existing { provider_id: i64 },
    New { input: ProviderEndpointInput },
}

#[derive(Debug, Deserialize)]
pub struct CreateCursorModels {
    pub provider: ProviderSelection,
    pub models: Vec<ProviderModelInput>,
}

#[derive(Debug, Serialize)]
pub struct CreatedCursorModels {
    pub provider: ProviderEndpoint,
    pub models: Vec<ProviderModel>,
}

#[derive(Debug, Deserialize)]
pub struct DiscoverCursorModels {
    pub provider: ProviderSelection,
}

pub async fn create(
    State(service): State<ControlService>,
    Json(input): Json<CreateCursorModels>,
) -> Result<(StatusCode, Json<CreatedCursorModels>)> {
    let (provider, models) = match input.provider {
        ProviderSelection::Existing { provider_id } => {
            let provider = service
                .providers()
                .await?
                .into_iter()
                .find(|provider| provider.provider_id == provider_id)
                .ok_or_else(|| crate::Error::RunNotFound(format!("provider {provider_id}")))?;
            let models = service.save_models(provider_id, &input.models).await?;
            (provider, models)
        }
        ProviderSelection::New { input: provider } => {
            service
                .create_provider_with_models(&provider, &input.models)
                .await?
        }
    };
    Ok((
        StatusCode::CREATED,
        Json(CreatedCursorModels { provider, models }),
    ))
}

pub async fn discover(
    State(service): State<ControlService>,
    Json(input): Json<DiscoverCursorModels>,
) -> Result<Json<DiscoveredModels>> {
    match input.provider {
        ProviderSelection::Existing { provider_id } => {
            Ok(Json(service.discover_models(provider_id).await?))
        }
        ProviderSelection::New { input } => Ok(Json(service.discover_input(&input).await?)),
    }
}
