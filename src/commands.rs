use sqlx::{PgPool, FromRow};

#[derive(FromRow, Debug, Clone)]
pub struct Command {
    pub guild_id: i64,
    pub name: String,
    pub response: String,
}

pub async fn get_command(pool: &PgPool, guild_id: i64, name: &str) -> Option<Command> {
    sqlx::query_as::<_, Command>("SELECT guild_id, name, response FROM commands WHERE guild_id = $1 AND name = $2")
        .bind(guild_id)
        .bind(name)
        .fetch_optional(pool)
        .await
        .ok()
        .flatten()
}

pub async fn add_command(pool: &PgPool, guild_id: i64, name: &str, response: &str) -> bool {
    sqlx::query("INSERT INTO commands (guild_id, name, response) VALUES ($1, $2, $3) ON CONFLICT (guild_id, name) DO NOTHING")
        .bind(guild_id)
        .bind(name)
        .bind(response)
        .execute(pool)
        .await
        .is_ok()
}

pub async fn update_command(pool: &PgPool, guild_id: i64, name: &str, response: &str) -> bool {
    sqlx::query("UPDATE commands SET response = $3 WHERE guild_id = $1 AND name = $2")
        .bind(guild_id)
        .bind(name)
        .bind(response)
        .execute(pool)
        .await
        .is_ok()
}

pub async fn remove_command(pool: &PgPool, guild_id: i64, name: &str) -> bool {
    sqlx::query("DELETE FROM commands WHERE guild_id = $1 AND name = $2")
        .bind(guild_id)
        .bind(name)
        .execute(pool)
        .await
        .is_ok()
}
