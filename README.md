# nkmzbot

Discord ボットに加えて、Axum 製の Web UI を追加しました。

## 必要な環境変数

- DISCORD_TOKEN: ボットトークン
- DATABASE_URL: Postgres 接続文字列
- WEB_BIND: Web サーバのバインドアドレス (例: `0.0.0.0:3000`、省略時はこの値)
- DISCORD_CLIENT_ID: Discord OAuth2 のクライアント ID
- DISCORD_CLIENT_SECRET: Discord OAuth2 のクライアントシークレット
- DISCORD_REDIRECT_URI: OAuth2 コールバック URL (例: `http://localhost:3000/oauth/callback`)
- SESSION_SECRET: セッション署名用のシークレット文字列 (ランダムな長い文字列推奨)

## 起動方法(ローカル)

- `.env` などで上記環境変数を設定
- `cargo run` で Bot と Web の両方が起動します
- ブラウザで `http://localhost:3000` にアクセス

## Web UI 機能

- Discord OAuth でログイン
- ログイン後、あなたが参加していて、かつ DB に登録済み(= commands テーブルにレコードがある)のギルド一覧を表示
- ギルドを選択すると、コマンド一覧の検索/追加/更新/一括削除が可能

## Docker

Docker で動かす場合、`WEB_BIND=0.0.0.0:3000` を必ず指定し、ポートを公開してください。

```bash
# 例
WEB_BIND=0.0.0.0:3000 \
DISCORD_TOKEN=... \
DATABASE_URL=postgres://... \
DISCORD_CLIENT_ID=... \
DISCORD_CLIENT_SECRET=... \
DISCORD_REDIRECT_URI=http://localhost:3000/oauth/callback \
SESSION_SECRET=$(openssl rand -hex 32) \
cargo run
```

> 注意: 現状は Cookie ベースの簡易セッションです。必要に応じてサーバサイドセッションに置き換えてください。
