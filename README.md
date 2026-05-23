# nkmzbot

Discord bot for managing custom commands.

## 必要な環境変数

- DISCORD_TOKEN: ボットトークン
- DATABASE_URL: Postgres 接続文字列
- API_URL: カスタムコマンド管理APIのURL
- API_TOKEN: カスタムコマンド管理APIのトークン
- IMM_BINARY: IMM CLIのパス（省略時 `imm`）。Docker Composeではホスト側のLinux IMMバイナリの絶対パス
- IMM_TIMEOUT_MS: IMM実行タイムアウトms（省略時 `3000`）
- IMM_MAX_SOURCE_BYTES: IMMソース上限（省略時 `65536`）
- IMM_MAX_OUTPUT_BYTES: IMM出力上限（省略時 `65536`）

## 起動方法(ローカル)

- `.env` などで上記環境変数を設定
- `go run cmd/nkmzbot/main.go` で Bot が起動します

## Docker

```bash
cp .env.sample .env
# .env の IMM_BINARY をサーバー上のLinux IMMバイナリの絶対パスにする
docker compose build
docker compose up -d
```

Docker Composeでは `.env` の `IMM_BINARY` をホスト側のバイナリとして
`/usr/local/bin/imm` にbind mountし、botには `IMM_BINARY=/usr/local/bin/imm`
として渡します。

## ビルド

```bash
go build -o nkmzbot cmd/nkmzbot/main.go
./nkmzbot
```

## Discord コマンド

ボットは以下のスラッシュコマンドを提供します：

### コマンド管理
- `/add` - 新しいコマンドを追加
  - `name`: コマンド名
  - `response`: 返答内容
- `/addbulk` - 複数のコマンドを一括で追加
  - `commands`: コマンドリスト（形式: `!cmd1: content1` 改行 `!cmd2: content2` ...）
- `/update` - 既存のコマンドを更新
  - `name`: コマンド名
  - `response`: 新しい返答内容
- `/remove` - コマンドを削除
  - `name`: コマンド名
- `/list` - 登録されているコマンド一覧を表示
- `/ramdom` - 登録されているコマンドからランダムに1つ返答
- `/imm run` - IMMコードをその場で実行
- `/imm check` - IMMコードを構文チェック
- `/imm command add|update|remove` - IMMカスタムコマンドを管理

### その他のコマンド
- `/jikan` - スケジュール実行を管理
- `/nomikai` - 飲み会割り勘セッションを操作
- `/guess` - ジオゲッサーを開始・プレイ
- `/wallet` - Wallet API を使って送金・請求を記録
- `Register as Response` - メッセージコンテキストメニュー（メッセージを右クリックしてコマンドとして登録）
- `Run as IMM` - メッセージコンテキストメニュー（メッセージ本文をIMMとして実行）

`/wallet` は以下のサブコマンドを提供します。

- `/wallet pay user amount memo?` - 自分が支払う送金を記録
- `/wallet req user amount memo?` - 相手への請求を記録

Wallet API への呼び出しでは既存の `API_TOKEN` を使い、操作ユーザーの Discord ID を `X-Discord-User-ID` ヘッダーで渡します。バックエンド側で Discord ID から Wallet 利用者を特定できる必要があります。

登録したコマンドは `!コマンド名` で呼び出せます。IMMコマンドは `!コマンド名 引数...` の形で引数を渡せます。IMM側では `bot_args`、`bot_raw`、`bot_user_id`、`bot_channel_id`、`bot_guild_id` を参照できます。

メッセージ右クリックからは以下を使えます。

- `Run as IMM`: 選択したメッセージ本文をIMMとして実行します。モーダルで引数を入力できます
- `Register as IMM`: 選択したメッセージ本文をIMMカスタムコマンドとして登録します
