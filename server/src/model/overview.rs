//! Read-only usage aggregates rendered by the desktop overview page.

use serde::Serialize;

#[derive(Clone, Debug, Default, PartialEq, Eq, Serialize)]
pub struct OverviewMetrics {
    pub llm_calls: i64,
    pub successful_calls: i64,
    pub failed_calls: i64,
    pub token_usage: i64,
    pub prompt_tokens: i64,
    pub input_tokens: i64,
    pub cache_read_tokens: i64,
    pub cache_write_tokens: i64,
    pub output_tokens: i64,
}

#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, Serialize)]
#[serde(rename_all = "snake_case")]
pub enum TokenUsageGranularity {
    Minute,
    Hour,
    #[default]
    Day,
}

#[derive(Clone, Debug, Default, PartialEq, Eq, Serialize)]
pub struct TokenUsageBucket {
    pub bucket_start_ms: i64,
    pub input_tokens: i64,
    pub cache_read_tokens: i64,
    pub cache_write_tokens: i64,
    pub output_tokens: i64,
}

impl TokenUsageBucket {
    pub fn total_tokens(&self) -> i64 {
        self.input_tokens
            .saturating_add(self.cache_read_tokens)
            .saturating_add(self.cache_write_tokens)
            .saturating_add(self.output_tokens)
    }
}

#[derive(Clone, Debug, Default, PartialEq, Eq, Serialize)]
pub struct Overview {
    pub metrics: OverviewMetrics,
    pub token_usage_granularity: TokenUsageGranularity,
    pub token_usage_series: Vec<TokenUsageBucket>,
}
