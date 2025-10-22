pub mod router;
pub mod oauth;
pub mod session;
pub mod templates;

use std::sync::Arc;
use axum::{Router, extract::FromRef};
use sqlx::PgPool;

#[derive(Clone)]
pub struct AppState {
    pub pool: Arc<PgPool>,
    pub discord_client_id: String,
    pub discord_client_secret: String,
    pub discord_redirect_uri: String,
    pub session_key: [u8; 32],
}

impl FromRef<AppState> for Arc<PgPool> {
    fn from_ref(state: &AppState) -> Arc<PgPool> {
        state.pool.clone()
    }
}

pub fn build_router(state: AppState) -> Router {
    router::create_router(state)
}
