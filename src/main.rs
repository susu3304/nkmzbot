use serenity::async_trait;
use serenity::model::{channel::Message, gateway::Ready};
use serenity::model::application::interaction::Interaction;
use serenity::model::application::command::CommandOptionType;
use serenity::prelude::*;
use sqlx::PgPool;
use futures::StreamExt;
use std::sync::Arc;
mod commands;

struct Handler {
    pool: Arc<PgPool>,
}

#[async_trait]
impl EventHandler for Handler {
    async fn ready(&self, ctx: Context, ready: Ready) {
        println!("{} is connected!", ready.user.name);
        
        // 全てのギルドにスラッシュコマンドを登録
        for guild in ready.guilds {
            let guild_id = guild.id;
            
            // addコマンド
            let _ = guild_id.create_application_command(&ctx.http, |command| {
                command
                    .name("add")
                    .description("新しいコマンドを追加します")
                    .create_option(|option| {
                        option
                            .name("name")
                            .description("コマンド名")
                            .kind(CommandOptionType::String)
                            .required(true)
                    })
                    .create_option(|option| {
                        option
                            .name("response")
                            .description("返答内容")
                            .kind(CommandOptionType::String)
                            .required(true)
                    })
            }).await;
            
            // removeコマンド
            let _ = guild_id.create_application_command(&ctx.http, |command| {
                command
                    .name("remove")
                    .description("コマンドを削除します")
                    .create_option(|option| {
                        option
                            .name("name")
                            .description("削除するコマンド名")
                            .kind(CommandOptionType::String)
                            .required(true)
                    })
            }).await;
            
            // updateコマンド
            let _ = guild_id.create_application_command(&ctx.http, |command| {
                command
                    .name("update")
                    .description("コマンドを更新します")
                    .create_option(|option| {
                        option
                            .name("name")
                            .description("更新するコマンド名")
                            .kind(CommandOptionType::String)
                            .required(true)
                    })
                    .create_option(|option| {
                        option
                            .name("response")
                            .description("新しい返答内容")
                            .kind(CommandOptionType::String)
                            .required(true)
                    })
            }).await;
            
            // listコマンド
            let _ = guild_id.create_application_command(&ctx.http, |command| {
                command
                    .name("list")
                    .description("登録されているコマンド一覧を表示します")
            }).await;
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

    async fn interaction_create(&self, ctx: Context, interaction: Interaction) {
        if let Interaction::ApplicationCommand(cmd) = interaction {
            let name = cmd.data.name.as_str();
            let guild_id = cmd.guild_id.map(|g| g.0 as i64);
            match name {
                "add" => {
                    if let Some(guild_id) = guild_id {
                        if cmd.data.options.len() >= 2 {
                            let cname = cmd.data.options[0].value.as_ref().and_then(|v| v.as_str()).unwrap_or("");
                            let resp = cmd.data.options[1].value.as_ref().and_then(|v| v.as_str()).unwrap_or("");
                            let ok = commands::add_command(&self.pool, guild_id, cname, resp).await;
                            let reply = if ok { format!("コマンド '{}' を追加しました。", cname) } else { "追加に失敗しました。".to_string() };
                            let _ = cmd.create_interaction_response(&ctx.http, |r| r.interaction_response_data(|d| d.content(reply))).await;
                        }
                    }
                },
                "remove" => {
                    if let Some(guild_id) = guild_id {
                        if !cmd.data.options.is_empty() {
                            let cname = cmd.data.options[0].value.as_ref().and_then(|v| v.as_str()).unwrap_or("");
                            let ok = commands::remove_command(&self.pool, guild_id, cname).await;
                            let reply = if ok { format!("コマンド '{}' を削除しました。", cname) } else { "そのコマンドは存在しません。".to_string() };
                            let _ = cmd.create_interaction_response(&ctx.http, |r| r.interaction_response_data(|d| d.content(reply))).await;
                        }
                    }
                },
                "update" => {
                    if let Some(guild_id) = guild_id {
                        if cmd.data.options.len() >= 2 {
                            let cname = cmd.data.options[0].value.as_ref().and_then(|v| v.as_str()).unwrap_or("");
                            let resp = cmd.data.options[1].value.as_ref().and_then(|v| v.as_str()).unwrap_or("");
                            let ok = commands::update_command(&self.pool, guild_id, cname, resp).await;
                            let reply = if ok { format!("コマンド '{}' を更新しました。", cname) } else { "そのコマンドは存在しません。".to_string() };
                            let _ = cmd.create_interaction_response(&ctx.http, |r| r.interaction_response_data(|d| d.content(reply))).await;
                        }
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
    dotenvy::dotenv().ok();
    let token = std::env::var("DISCORD_TOKEN").expect("Expected a token in the environment");
    let database_url = std::env::var("DATABASE_URL").expect("Expected a database url in the environment");
    
    // DB接続とマイグレーション実行
    let pool = PgPool::connect(&database_url).await.expect("DB接続失敗");
    
    // マイグレーション実行
    println!("Running database migrations...");
    if let Err(e) = sqlx::migrate!("./migrations").run(&pool).await {
        eprintln!("Migration failed: {}", e);
        std::process::exit(1);
    }
    println!("Migrations completed successfully!");
    
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
