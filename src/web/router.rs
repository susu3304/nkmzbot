use std::sync::Arc;

use axum::{routing::{get, post}, Router, extract::{Path, Query, State}, response::{Html, IntoResponse, Redirect}, Form, http::StatusCode};
use askama::Template;
use serde::Deserialize;
use sqlx::Row;

use super::{AppState};
use crate::web::{oauth, session};

pub fn create_router(state: AppState) -> Router {
    Router::new()
        .route("/", get(home))
        .route("/login", get(oauth::login))
        .route("/oauth/callback", get(oauth::oauth_callback))
        .route("/logout", get(oauth::logout))
        .route("/dashboard", get(dashboard))
        .route("/guilds/:guild_id/commands", get(commands_page))
        .route("/guilds/:guild_id/commands/add", post(add_command))
        .route("/guilds/:guild_id/commands/update", post(update_command))
        .route("/guilds/:guild_id/commands/bulk-delete", post(bulk_delete_commands))
        .with_state(state)
}

async fn home(State(_state): State<AppState>, jar: axum_extra::extract::cookie::CookieJar) -> impl IntoResponse {
    if jar.get("session").is_some() {
        // CookieJarは変更なしでそのまま返す
        return (jar, Redirect::to("/dashboard")).into_response();
    }
    // CSRFトークンを発行
    use rand::{distributions::Alphanumeric, Rng};
    let csrf: String = rand::thread_rng().sample_iter(&Alphanumeric).take(32).map(char::from).collect();
    let mut cookie = axum_extra::extract::cookie::Cookie::new("csrf", csrf);
    cookie.set_http_only(true);
    cookie.set_same_site(axum_extra::extract::cookie::SameSite::Lax);
    cookie.set_path("/");
    let jar = jar.add(cookie);
    let tpl = crate::web::templates::HomeTemplate {};
    (jar, Html(tpl.render().unwrap_or_else(|_| "<h1>Home</h1>".to_string()))).into_response()
}

async fn dashboard(State(state): State<AppState>, jar: axum_extra::extract::cookie::CookieJar) -> impl IntoResponse {
    let Some(sealed) = jar.get("session").map(|c| c.value().to_string()) else {
        return Redirect::to("/").into_response();
    };
    let Some(access_token) = session::open_token(&state.session_key, &sealed) else {
        return Redirect::to("/").into_response();
    };

    // ユーザGuild取得
    let guilds = match oauth::fetch_user_guilds(&access_token).await {
        Ok(v) => v,
        Err(_) => return (StatusCode::BAD_GATEWAY, "Failed to fetch guilds").into_response(),
    };
    // DB登録済みguild_idの集合
    let pool: Arc<sqlx::PgPool> = state.pool.clone();
    let rows = match sqlx::query("SELECT DISTINCT guild_id FROM commands")
        .fetch_all(&*pool)
        .await
    {
        Ok(r) => r,
        Err(_) => vec![],
    };
    let db_guilds: std::collections::HashSet<i64> = rows
        .into_iter()
        .filter_map(|row| row.try_get::<i64, _>("guild_id").ok())
        .collect();

    // フィルタリング
    let filtered: Vec<crate::web::oauth::DiscordGuild> = guilds
        .into_iter()
        .filter(|g| db_guilds.contains(&g.id.parse::<i64>().unwrap_or_default()))
        .collect();

    let username = jar.get("username").map(|c| c.value().to_string());
    let tpl = crate::web::templates::DashboardTemplate { username, guilds: filtered };
    Html(tpl.render().unwrap()).into_response()
}

#[derive(Debug, Deserialize)]
struct ListQuery { q: Option<String> }

pub async fn commands_page(
    State(state): State<AppState>,
    jar: axum_extra::extract::cookie::CookieJar,
    Path(guild_id): Path<i64>,
    Query(ListQuery { q }): Query<ListQuery>,
) -> impl IntoResponse {
    // 認証チェック
    let Some(sealed) = jar.get("session").map(|c| c.value().to_string()) else { return Redirect::to("/").into_response(); };
    let Some(access_token) = session::open_token(&state.session_key, &sealed) else { return Redirect::to("/").into_response(); };
    // 所属ギルドか検証
    match oauth::fetch_user_guilds(&access_token).await {
        Ok(gs) => {
            let ok = gs.iter().any(|g| g.id.parse::<i64>().ok() == Some(guild_id));
            if !ok { return Redirect::to("/").into_response(); }
        }
        Err(_) => return (StatusCode::BAD_GATEWAY, "Failed to fetch guilds").into_response(),
    }

    let pattern = q.clone().unwrap_or_default();
    let like = if pattern.is_empty() { None } else { Some(format!("%{}%", pattern)) };

    #[derive(sqlx::FromRow, Clone)]
    struct RowCmd { name: String, response: String }

    let sql = if like.is_some() {
        "SELECT name, response FROM commands WHERE guild_id=$1 AND (name ILIKE $2 OR response ILIKE $2) ORDER BY name"
    } else {
        "SELECT name, response FROM commands WHERE guild_id=$1 ORDER BY name"
    };

    let pool = state.pool.clone();
    let cmds: Vec<RowCmd> = if let Some(like_s) = like.as_deref() {
        sqlx::query_as::<_, RowCmd>(sql)
            .bind(guild_id)
            .bind(like_s)
            .fetch_all(&*pool)
            .await
            .unwrap_or_default()
    } else {
        sqlx::query_as::<_, RowCmd>(sql)
            .bind(guild_id)
            .fetch_all(&*pool)
            .await
            .unwrap_or_default()
    };

    let csrf = jar.get("csrf").map(|c| c.value().to_string()).unwrap_or_default();
    let converted = cmds.into_iter().map(|c| crate::web::templates::CmdRow { name: c.name, response: c.response }).collect();
    let tpl = crate::web::templates::CommandsTemplate { guild_id, q: q.unwrap_or_default(), commands: converted, csrf };
    Html(tpl.render().unwrap()).into_response()
}

#[derive(Debug, Deserialize)]
struct AddForm { name: String, response: String, csrf: String }

async fn add_command(State(state): State<AppState>, jar: axum_extra::extract::cookie::CookieJar, Path(guild_id): Path<i64>, Form(f): Form<AddForm>) -> impl IntoResponse {
    if jar.get("csrf").map(|c| c.value()) != Some(f.csrf.as_str()) { return (StatusCode::BAD_REQUEST, "invalid csrf").into_response(); }
    // 認可チェック
    let Some(sealed) = jar.get("session").map(|c| c.value().to_string()) else { return Redirect::to("/").into_response(); };
    let Some(access_token) = session::open_token(&state.session_key, &sealed) else { return Redirect::to("/").into_response(); };
    if let Ok(gs) = oauth::fetch_user_guilds(&access_token).await {
        let ok = gs.iter().any(|g| g.id.parse::<i64>().ok() == Some(guild_id));
        if !ok { return Redirect::to("/").into_response(); }
    }
    let ok = crate::commands::add_command(&state.pool, guild_id, &f.name, &f.response).await;
    let to = format!("/guilds/{guild_id}/commands");
    if ok { Redirect::to(&to).into_response() } else { (StatusCode::BAD_REQUEST, "failed to add").into_response() }
}

#[derive(Debug, Deserialize)]
struct UpdateForm { name: String, response: String, csrf: String }

async fn update_command(State(state): State<AppState>, jar: axum_extra::extract::cookie::CookieJar, Path(guild_id): Path<i64>, Form(f): Form<UpdateForm>) -> impl IntoResponse {
    if jar.get("csrf").map(|c| c.value()) != Some(f.csrf.as_str()) { return (StatusCode::BAD_REQUEST, "invalid csrf").into_response(); }
    let Some(sealed) = jar.get("session").map(|c| c.value().to_string()) else { return Redirect::to("/").into_response(); };
    let Some(access_token) = session::open_token(&state.session_key, &sealed) else { return Redirect::to("/").into_response(); };
    if let Ok(gs) = oauth::fetch_user_guilds(&access_token).await {
        let ok = gs.iter().any(|g| g.id.parse::<i64>().ok() == Some(guild_id));
        if !ok { return Redirect::to("/").into_response(); }
    }
    let ok = crate::commands::update_command(&state.pool, guild_id, &f.name, &f.response).await;
    let to = format!("/guilds/{guild_id}/commands");
    if ok { Redirect::to(&to).into_response() } else { (StatusCode::BAD_REQUEST, "failed to update").into_response() }
}

#[derive(Debug, Deserialize)]
struct BulkDeleteForm { names: Option<Vec<String>>, csrf: String }

async fn bulk_delete_commands(State(state): State<AppState>, jar: axum_extra::extract::cookie::CookieJar, Path(guild_id): Path<i64>, Form(f): Form<BulkDeleteForm>) -> impl IntoResponse {
    if jar.get("csrf").map(|c| c.value()) != Some(f.csrf.as_str()) { return (StatusCode::BAD_REQUEST, "invalid csrf").into_response(); }
    let Some(sealed) = jar.get("session").map(|c| c.value().to_string()) else { return Redirect::to("/").into_response(); };
    let Some(access_token) = session::open_token(&state.session_key, &sealed) else { return Redirect::to("/").into_response(); };
    if let Ok(gs) = oauth::fetch_user_guilds(&access_token).await {
        let ok = gs.iter().any(|g| g.id.parse::<i64>().ok() == Some(guild_id));
        if !ok { return Redirect::to("/").into_response(); }
    }
    if let Some(names) = f.names {
        for name in names {
            let _ = crate::commands::remove_command(&state.pool, guild_id, &name).await;
        }
    }
    Redirect::to(&format!("/guilds/{guild_id}/commands")).into_response()
}
