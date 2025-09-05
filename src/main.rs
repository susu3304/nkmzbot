use serenity::async_trait;
use serenity::model::{channel::Message, gateway::Ready, interactions::application_command::{ApplicationCommandInteraction, CommandDataOptionValue}};
use serenity::prelude::*;
use sqlx::PgPool;
use futures::stream::StreamExt;
use std::sync::Arc;
mod commands;

struct Handler {
    pool: Arc<PgPool>,
}

#[async_trait]
impl EventHandler for Handler {
    async fn ready(&self, _ctx: Context, ready: Ready) {
        println!("{} is connected!", ready.user.name);
        // スラッシュコマンド登録
        let guilds = ready.guilds;
        for guild in guilds {
            let _ = _ctx.http.create_guild_application_command(guild.id.0, |cmd| {
                cmd.name("add")
                    .description("コマンドを追加")
                    .create_option(|opt| {
                        opt.name("name")
                            .kind(serenity::model::interactions::application_command::ApplicationCommandOptionType::String)
                            .required(true)
                            .description("コマンド名")
                    })
                    .create_option(|opt| {
                        opt.name("response")
                            .kind(serenity::model::interactions::application_command::ApplicationCommandOptionType::String)
                            .required(true)
                            .description("返答文")
                    })
            });
            let _ = _ctx.http.create_guild_application_command(guild.id.0, |cmd| {
                cmd.name("remove")
                    .description("コマンドを削除")
                    .create_option(|opt| {
                        opt.name("name")
                            .kind(serenity::model::interactions::application_command::ApplicationCommandOptionType::String)
                            .required(true)
                            .description("コマンド名")
                    })
            });
            let _ = _ctx.http.create_guild_application_command(guild.id.0, |cmd| {
                cmd.name("update")
                    .description("コマンドを更新")
                    .create_option(|opt| {
                        opt.name("name")
                            .kind(serenity::model::interactions::application_command::ApplicationCommandOptionType::String)
                            .required(true)
                            .description("コマンド名")
                    })
                    .create_option(|opt| {
                        opt.name("response")
                            .kind(serenity::model::interactions::application_command::ApplicationCommandOptionType::String)
                            .required(true)
                            .description("新しい返答文")
                    })
            });
            let _ = _ctx.http.create_guild_application_command(guild.id.0, |cmd| {
                cmd.name("list")
                    .description("コマンド一覧を表示")
            });
        }
    }

    async fn message(&self, ctx: Context, msg: Message) {
        let content = msg.content.trim();
        // 通常コマンドのみテキストで応答
        if content.starts_with('!') && content.len() > 1 {
            let cmd = &content[1..];
            let guild_id = msg.guild_id.map(|g| g.0 as i64);
            if let Some(guild_id) = guild_id {
                if let Some(command) = commands::get_command(&self.pool, guild_id, cmd).await {
                    let _ = msg.reply(&ctx, &command.response).await;
                }
            }
        }
    }

    async fn interaction_create(&self, ctx: Context, interaction: serenity::model::prelude::Interaction) {
        if let serenity::model::prelude::Interaction::ApplicationCommand(cmd) = interaction {
            let name = cmd.data.name.as_str();
            let guild_id = cmd.guild_id.map(|g| g.0 as i64);
            match name {
                "add" => {
                    if let Some(guild_id) = guild_id {
                        let cname = cmd.data.options.get(0).and_then(|o| match &o.value { CommandDataOptionValue::String(s) => Some(s), _ => None }).unwrap();
                        let resp = cmd.data.options.get(1).and_then(|o| match &o.value { CommandDataOptionValue::String(s) => Some(s), _ => None }).unwrap();
                        let ok = commands::add_command(&self.pool, guild_id, cname, resp).await;
                        let reply = if ok { format!("コマンド '{}' を追加しました。", cname) } else { "追加に失敗しました。".to_string() };
                        let _ = cmd.create_interaction_response(&ctx.http, |r| r.interaction_response_data(|d| d.content(reply))).await;
                    }
                },
                "remove" => {
                    if let Some(guild_id) = guild_id {
                        let cname = cmd.data.options.get(0).and_then(|o| match &o.value { CommandDataOptionValue::String(s) => Some(s), _ => None }).unwrap();
                        let ok = commands::remove_command(&self.pool, guild_id, cname).await;
                        let reply = if ok { format!("コマンド '{}' を削除しました。", cname) } else { "そのコマンドは存在しません。".to_string() };
                        let _ = cmd.create_interaction_response(&ctx.http, |r| r.interaction_response_data(|d| d.content(reply))).await;
                    }
                },
                "update" => {
                    if let Some(guild_id) = guild_id {
                        let cname = cmd.data.options.get(0).and_then(|o| match &o.value { CommandDataOptionValue::String(s) => Some(s), _ => None }).unwrap();
                        let resp = cmd.data.options.get(1).and_then(|o| match &o.value { CommandDataOptionValue::String(s) => Some(s), _ => None }).unwrap();
                        let ok = commands::update_command(&self.pool, guild_id, cname, resp).await;
                        let reply = if ok { format!("コマンド '{}' を更新しました。", cname) } else { "そのコマンドは存在しません。".to_string() };
                        let _ = cmd.create_interaction_response(&ctx.http, |r| r.interaction_response_data(|d| d.content(reply))).await;
                    }
                },
                "list" => {
                    if let Some(guild_id) = guild_id {
                        let mut rows = sqlx::query_as::<_, commands::Command>("SELECT guild_id, name, response FROM commands WHERE guild_id = $1 ORDER BY name")
                            .bind(guild_id)
                            .fetch(&*self.pool);
                        let mut entries = Vec::new();
                        while let Some(Ok(cmd)) = rows.next().await {
                            entries.push(format!("!{}: {}", cmd.name, cmd.response));
                        }
                        if entries.is_empty() {
                            let _ = cmd.create_interaction_response(&ctx.http, |r| r.interaction_response_data(|d| d.content("コマンドは登録されていません。"))).await;
                            return;
                        }
                        // 2000文字制限で分割送信
                        let mut buffer = String::new();
                        for entry in entries {
                            if buffer.len() + entry.len() + 1 > 2000 {
                                let _ = cmd.channel_id.say(&ctx.http, buffer.clone()).await;
                                buffer.clear();
                            }
                            if !buffer.is_empty() {
                                buffer.push('\n');
                            }
                            buffer.push_str(&entry);
                        }
                        if !buffer.is_empty() {
                            let _ = cmd.channel_id.say(&ctx.http, buffer).await;
                        }
                        let _ = cmd.create_interaction_response(&ctx.http, |r| r.interaction_response_data(|d| d.content("コマンド一覧を送信しました。"))).await;
                    }
                },
                _ => {}
            }
        }
    }
}

#[tokio::main]
async fn main() {
    dotenv::dotenv().ok();
    let token = std::env::var("DISCORD_TOKEN").expect("Expected a token in the environment");
    let database_url = std::env::var("DATABASE_URL").expect("Expected a database url in the environment");
    let pool = PgPool::connect(&database_url).await.expect("DB接続失敗");
    let handler = Handler { pool: Arc::new(pool) };
    let intents = GatewayIntents::all();
    let mut client = Client::builder(&token, intents)
        .event_handler(handler)
        .await
        .expect("Error creating client");

    if let Err(why) = client.start().await {
        println!("Client error: {:?}", why);
    }
}
