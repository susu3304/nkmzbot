use serenity::async_trait;
use serenity::model::{channel::Message, gateway::Ready};
use serenity::model::application::interaction::Interaction;
use serenity::model::application::command::CommandOptionType;
use serenity::model::application::command::CommandType;
use serenity::model::application::component::ActionRowComponent;
use serenity::model::application::component::InputTextStyle;
use serenity::model::guild::Guild;
use serenity::model::id::GuildId;
use serenity::prelude::*;
use sqlx::PgPool;
use futures::StreamExt;
use std::sync::Arc;
use axum::Router;
use axum_extra::extract::cookie::{Cookie, SameSite};
use tokio::task::JoinSet;
mod web;
mod commands;

struct Handler {
    pool: Arc<PgPool>,
}

async fn register_guild_commands(ctx: &Context, guild_id: GuildId) {
    // ギルド内のアプリケーションコマンドを「置き換え」る（重複防止）
    if let Err(e) = guild_id
        .set_application_commands(&ctx.http, |commands| {
            commands
                .create_application_command(|command| {
                    command
                        .name("add")
                        .description("新しいコマンドを追加します")
                        .dm_permission(false)
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
                })
                .create_application_command(|command| {
                    command
                        .name("remove")
                        .description("コマンドを削除します")
                        .dm_permission(false)
                        .create_option(|option| {
                            option
                                .name("name")
                                .description("削除するコマンド名")
                                .kind(CommandOptionType::String)
                                .required(true)
                        })
                })
                .create_application_command(|command| {
                    command
                        .name("update")
                        .description("コマンドを更新します")
                        .dm_permission(false)
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
                })
                .create_application_command(|command| {
                    command
                        .name("list")
                        .description("登録されているコマンド一覧を表示します")
                        .dm_permission(false)
                })
                .create_application_command(|command| {
                    command
                        .name("Register as Response")
                        .kind(CommandType::Message)
                })
        })
        .await
    {
        eprintln!(
            "Failed to register application commands for guild {}: {:?}",
            guild_id.0, e
        );
    } else {
        println!("Registered application commands for guild {}", guild_id.0);
    }
}

#[async_trait]
impl EventHandler for Handler {
    async fn ready(&self, ctx: Context, ready: Ready) {
        println!("{} is connected!", ready.user.name);
        
        // 既存の全ギルドにスラッシュコマンドを登録（重複防止のため置換）
        for guild in ready.guilds {
            register_guild_commands(&ctx, guild.id).await;
        }
    }

    async fn guild_create(&self, ctx: Context, guild: Guild) {
        // ギルドが作成/利用可能になったら、コマンドを確実に登録
        println!("Guild available/joined: {} (id={}) — ensuring commands", guild.name, guild.id.0);
        register_guild_commands(&ctx, guild.id).await;
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
        match interaction {
            Interaction::ApplicationCommand(cmd) => {
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
                    "Register as Response" => {
                        if let Some(guild_id) = guild_id {
                            println!("Processing Register as Response command");
                            // メッセージコンテキストメニューからの場合
                            if !cmd.data.resolved.messages.is_empty() {
                                println!("Found {} messages", cmd.data.resolved.messages.len());
                                if let Some((_, message)) = cmd.data.resolved.messages.iter().next() {
                                    println!("Processing message: {}", message.content);
                                    let message_content = &message.content;
                                    println!("Processing message: {}", message_content);
                                    // メッセージIDをcustom_idに使用
                                    let custom_id = format!("reg_resp:{}", message.id.0);
                                    
                                    // モーダルでコマンド名を入力してもらう
                                    match cmd.create_interaction_response(&ctx.http, |response| {
                                        response
                                            .kind(serenity::model::prelude::InteractionResponseType::Modal)
                                            .interaction_response_data(|data| {
                                                data.custom_id(&custom_id)
                                                    .title("コマンド名を入力")
                                                    .components(|components| {
                                                        components.create_action_row(|row| {
                                                            row.create_input_text(|input| {
                                                                input
                                                                    .custom_id("command_name")
                                                                    .label("コマンド名")
                                                                    .placeholder("例: hello")
                                                                    .required(true)
                                                                    .max_length(50)
                                                                    .style(InputTextStyle::Short)
                                                            })
                                                        })
                                                    })
                                            })
                                    }).await {
                                        Ok(_) => println!("Modal created successfully"),
                                        Err(e) => println!("Failed to create modal: {:?}", e),
                                    }
                                } else {
                                    let _ = cmd.create_interaction_response(&ctx.http, |response| {
                                        response
                                            .kind(serenity::model::prelude::InteractionResponseType::ChannelMessageWithSource)
                                            .interaction_response_data(|message| {
                                                message.content("メッセージが見つかりませんでした。")
                                            })
                                    }).await;
                                }
                            } else {
                                let _ = cmd.create_interaction_response(&ctx.http, |response| {
                                    response
                                        .kind(serenity::model::prelude::InteractionResponseType::ChannelMessageWithSource)
                                        .interaction_response_data(|message| {
                                            message.content("メッセージが見つかりませんでした。")
                                        })
                                }).await;
                            }
                        }
                    },
                    _ => {}
                }
            },
            Interaction::ModalSubmit(modal) => {
                if modal.data.custom_id.starts_with("reg_resp:") {
                    if let Some(guild_id) = modal.guild_id.map(|g| g.0 as i64) {
                        // custom_idからメッセージIDを取得
                        let message_id_str = &modal.data.custom_id[9..]; // "reg_resp:"の後
                        if let Ok(message_id) = message_id_str.parse::<u64>() {
                            let message_id = serenity::model::id::MessageId(message_id);
                            // メッセージを再取得
                            if let Ok(message) = modal.channel_id.message(&ctx.http, message_id).await {
                                if let Some(action_row) = modal.data.components.get(0) {
                                    if let Some(component) = action_row.components.get(0) {
                                        if let ActionRowComponent::InputText(input) = component {
                                            let command_name = &input.value;
                                            
                                            // メッセージ内容を構築（添付ファイルがある場合はURLを追加）
                                            let mut response_content = message.content.clone();
                                            
                                            // 添付ファイルがある場合はURLを追加
                                            if !message.attachments.is_empty() {
                                                for attachment in &message.attachments {
                                                    // すべての添付ファイルを追加（画像のみにしたい場合は下の行のコメントアウトを外す）
                                                    // if attachment.content_type.as_ref().map_or(false, |ct| ct.starts_with("image/")) {
                                                        if !response_content.is_empty() {
                                                            response_content.push('\n');
                                                        }
                                                        response_content.push_str(&attachment.url);
                                                    // }
                                                }
                                            }
                                            
                                            // メッセージ内容をコマンドの返答として登録
                                            let ok = commands::add_command(&self.pool, guild_id, command_name, &response_content).await;
                                            let reply = if ok {
                                                format!("メッセージの内容をコマンド '{}' の返答として登録しました！", command_name)
                                            } else {
                                                "登録に失敗しました。同じ名前のコマンドが既に存在するかもしれません。".to_string()
                                            };
                                            let _ = modal.create_interaction_response(&ctx.http, |r| {
                                                r.interaction_response_data(|d| {
                                                    d.content(reply)
                                                })
                                            }).await;
                                        }
                                    }
                                }
                            } else {
                                let _ = modal.create_interaction_response(&ctx.http, |r| {
                                    r.interaction_response_data(|d| {
                                        d.content("メッセージの取得に失敗しました。")
                                    })
                                }).await;
                            }
                        }
                    }
                }
            },
            _ => {}
        }
    }
}

#[tokio::main]
async fn main() {
    dotenvy::dotenv().ok();
    let token = std::env::var("DISCORD_TOKEN").expect("Expected a token in the environment");
    let database_url = std::env::var("DATABASE_URL").expect("Expected a database url in the environment");
    let web_bind = std::env::var("WEB_BIND").unwrap_or_else(|_| "0.0.0.0:3000".to_string());
    let discord_client_id = std::env::var("DISCORD_CLIENT_ID").expect("DISCORD_CLIENT_ID not set");
    let discord_client_secret = std::env::var("DISCORD_CLIENT_SECRET").expect("DISCORD_CLIENT_SECRET not set");
    let discord_redirect_uri = std::env::var("DISCORD_REDIRECT_URI").unwrap_or_else(|_| "http://localhost:3000/oauth/callback".to_string());
    let session_secret = std::env::var("SESSION_SECRET").unwrap_or_else(|_| "dev-only-change-me".to_string());
    
    // DB接続とマイグレーション実行
    let pool = PgPool::connect(&database_url).await.expect("DB接続失敗");
    
    // マイグレーション実行
    println!("Running database migrations...");
    if let Err(e) = sqlx::migrate!("./migrations").run(&pool).await {
        eprintln!("Migration failed: {}", e);
        std::process::exit(1);
    }
    println!("Migrations completed successfully!");
    
    let pool = Arc::new(pool);
    let handler = Handler { pool: pool.clone() };
    let intents = GatewayIntents::all();
    let mut client = Client::builder(&token, intents)
        .event_handler(handler)
        .await
        .expect("Error creating client");

    // Web state 構築
    let session_key = web::session::derive_key_from_env(&session_secret);
    let state = web::AppState {
        pool: pool.clone(),
        discord_client_id,
        discord_client_secret,
        discord_redirect_uri,
        session_key,
    };
    let app: Router = web::build_router(state);

    let mut set = JoinSet::new();
    // Discord Bot
    set.spawn(async move {
        if let Err(why) = client.start().await {
            eprintln!("Client error: {:?}", why);
        }
    });
    // Web server
    set.spawn(async move {
        use axum::routing::get;
        use axum::http::StatusCode;
        let listener = tokio::net::TcpListener::bind(&web_bind).await.expect("bind web");
        println!("Web listening on http://{}", web_bind);
        axum::serve(listener, app)
            .with_graceful_shutdown(async {
                // TODO: hook shutdown signals
            })
            .await
            .map_err(|e| eprintln!("web server error: {e}"))
            .ok();
    });

    // プロセスを維持
    while let Some(_res) = set.join_next().await {}
}
