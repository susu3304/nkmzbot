use axum::{extract::{Query, State}, response::{IntoResponse, Redirect}, http::StatusCode};
use axum_extra::extract::cookie::{Cookie, CookieJar, SameSite};
use rand::{distributions::Alphanumeric, Rng};
use reqwest::Client;
use serde::{Deserialize, Serialize};

use super::AppState;

const DISCORD_AUTH_URL: &str = "https://discord.com/api/oauth2/authorize";
const DISCORD_TOKEN_URL: &str = "https://discord.com/api/oauth2/token";
const DISCORD_API_BASE: &str = "https://discord.com/api";

#[derive(Debug, Deserialize)]
pub struct AuthQuery {
    code: Option<String>,
    state: Option<String>,
    error: Option<String>,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct DiscordUser {
    pub id: String,
    pub username: String,
    pub global_name: Option<String>,
    pub avatar: Option<String>,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct DiscordGuild {
    pub id: String,
    pub name: String,
    pub owner: Option<bool>,
}

#[derive(Debug, Serialize, Deserialize)]
struct TokenResponse {
    access_token: String,
    token_type: String,
    expires_in: i64,
    refresh_token: Option<String>,
    scope: String,
}

pub async fn login(State(state): State<super::AppState>, jar: CookieJar) -> impl IntoResponse {
    let state_token: String = rand::thread_rng()
        .sample_iter(&Alphanumeric)
        .take(32)
        .map(char::from)
        .collect();

    let mut cookie = Cookie::new("oauth_state", state_token.clone());
    cookie.set_http_only(true);
    cookie.set_same_site(SameSite::Lax);
    cookie.set_path("/");
    let jar = jar.add(cookie);

    let url = format!(
        "{}?client_id={}&response_type=code&scope=identify%20guilds&redirect_uri={}&state={}",
        DISCORD_AUTH_URL,
        urlencoding::encode(&state.discord_client_id),
        urlencoding::encode(&state.discord_redirect_uri),
        urlencoding::encode(&state_token)
    );
    (jar, Redirect::to(&url))
}

pub async fn oauth_callback(
    State(state): State<AppState>,
    jar: CookieJar,
    Query(q): Query<AuthQuery>,
) -> impl IntoResponse {
    if let Some(err) = q.error.as_deref() {
        return (StatusCode::BAD_REQUEST, format!("OAuth error: {}", err)).into_response();
    }
    let Some(code) = q.code else {
        return (StatusCode::BAD_REQUEST, "missing code").into_response();
    };
    let Some(state_param) = q.state else {
        return (StatusCode::BAD_REQUEST, "missing state").into_response();
    };
    let Some(cookie_state) = jar.get("oauth_state").map(|c| c.value().to_string()) else {
        return (StatusCode::BAD_REQUEST, "missing oauth_state cookie").into_response();
    };
    if cookie_state != state_param {
        return (StatusCode::BAD_REQUEST, "invalid state").into_response();
    }

    let client = Client::new();
    let form = [
        ("client_id", state.discord_client_id.as_str()),
        ("client_secret", state.discord_client_secret.as_str()),
        ("grant_type", "authorization_code"),
        ("code", code.as_str()),
        ("redirect_uri", state.discord_redirect_uri.as_str()),
        // Discord では scope は省略可能だが、念のため明示する
        ("scope", "identify guilds"),
    ];
    let resp = match client.post(DISCORD_TOKEN_URL).form(&form).send().await {
        Ok(r) => r,
        Err(e) => return (StatusCode::BAD_GATEWAY, format!("token exchange failed: {e}")).into_response(),
    };
    let resp = match resp.error_for_status() {
        Ok(r) => r,
        Err(e) => return (StatusCode::BAD_GATEWAY, format!("token exchange failed: {e}")).into_response(),
    };
    let token_res: TokenResponse = match resp.json().await {
        Ok(t) => t,
        Err(e) => return (StatusCode::BAD_GATEWAY, format!("token parse failed: {e}")).into_response(),
    };

    // ユーザ情報取得
    let resp = match client
        .get(format!("{}/users/@me", DISCORD_API_BASE))
        .bearer_auth(&token_res.access_token)
        .send()
        .await
    {
        Ok(r) => r,
        Err(e) => return (StatusCode::BAD_GATEWAY, format!("get user failed: {e}")).into_response(),
    };
    let resp = match resp.error_for_status() {
        Ok(r) => r,
        Err(e) => return (StatusCode::BAD_GATEWAY, format!("get user failed: {e}")).into_response(),
    };
    let user: DiscordUser = match resp.json().await {
        Ok(u) => u,
        Err(e) => return (StatusCode::BAD_GATEWAY, format!("user parse failed: {e}")).into_response(),
    };

    // アクセストークンをセッションCookieに格納
    let value = crate::web::session::seal_token(&state.session_key, &token_res.access_token);
    let mut sess = Cookie::new("session", value);
    sess.set_http_only(true);
    sess.set_same_site(SameSite::Lax);
    sess.set_path("/");
    let jar = jar.add(sess);

    // ユーザ名を軽く表示用にCookieに
    let mut u = Cookie::new("username", user.global_name.unwrap_or(user.username));
    u.set_same_site(SameSite::Lax);
    u.set_path("/");
    let jar = jar.add(u);

    (jar, Redirect::to("/dashboard")).into_response()
}

pub async fn logout(jar: CookieJar) -> impl IntoResponse {
    // 削除時も Path=/ を指定して確実に削除
    let mut s = Cookie::from("session");
    s.set_path("/");
    let jar = jar.remove(s);
    let mut u = Cookie::from("username");
    u.set_path("/");
    let jar = jar.remove(u);
    (jar, Redirect::to("/"))
}

pub async fn fetch_user_guilds(access_token: &str) -> Result<Vec<DiscordGuild>, reqwest::Error> {
    let client = Client::new();
    let res = client
        .get(format!("{}/users/@me/guilds", DISCORD_API_BASE))
        .bearer_auth(access_token)
        .header(reqwest::header::USER_AGENT, "nkmzbot/1.0 (+https://github.com/susu3304/nkmzbot)")
        .header(reqwest::header::ACCEPT, "application/json")
        .send()
        .await?;

    let status = res.status();
    let res = match res.error_for_status() {
        Ok(r) => r,
        Err(e) => {
            eprintln!("[oauth] fetch_user_guilds failed: status={} err={}", status, e);
            return Err(e);
        }
    };

    let guilds: Vec<DiscordGuild> = res.json().await?;
    Ok(guilds)
}
