# デプロイ設定手順

このプロジェクトは、GitHub Actionsを使用してmainブランチへのマージ時に自動デプロイされます。

## 1. GitHub Secretsの設定

リポジトリの Settings → Secrets and variables → Actions → New repository secret から以下を設定してください：

### 必須のSecrets

| Secret名 | 説明 | 例 |
|---------|------|-----|
| `SSH_HOST` | デプロイ先サーバーのホスト名またはIPアドレス | `192.168.1.100` または `example.com` |
| `SSH_USER` | SSHログインユーザー名 | `wabu` |
| `SSH_PRIVATE_KEY` | SSH秘密鍵（デプロイキー） | 下記参照 |

### オプションのSecrets

| Secret名 | 説明 | デフォルト値 |
|---------|------|------------|
| `SSH_PORT` | SSHポート番号 | `22` |
| `DEPLOY_PATH` | デプロイ先のディレクトリパス | `~/workspace/nkmzbot` |

## 2. SSH秘密鍵の生成と設定

### サーバー側での作業

```bash
# デプロイ用のSSH鍵ペアを生成（サーバー上で実行）
ssh-keygen -t ed25519 -C "github-deploy" -f ~/.ssh/github_deploy_key

# 公開鍵を authorized_keys に追加
cat ~/.ssh/github_deploy_key.pub >> ~/.ssh/authorized_keys

# 秘密鍵の内容を表示（コピーしてGitHub Secretsに設定）
cat ~/.ssh/github_deploy_key
```

### GitHub側での作業

1. 上記で表示された秘密鍵の内容を**すべて**コピー（`-----BEGIN OPENSSH PRIVATE KEY-----` から `-----END OPENSSH PRIVATE KEY-----` まで）
2. GitHubリポジトリの Settings → Secrets → New repository secret
3. Name: `SSH_PRIVATE_KEY`
4. Value: コピーした秘密鍵を貼り付け
5. Add secret をクリック

## 3. サーバー側の `.env`

`docker-compose.yml` は `.env` の `IMM_BINARY` をホスト側のLinux IMMバイナリとして
コンテナ内の `/usr/local/bin/imm` にbind mountします。

```env
IMM_BINARY=/usr/local/bin/imm
```

IMMを別の場所に入れている場合は、その絶対パスを指定してください。

```env
IMM_BINARY=/home/wabu/.cargo/bin/imm-native
```

## 4. デプロイフロー

```
mainブランチにマージ
  ↓
GitHub Actions 起動
  ↓
SSH経由でサーバーに接続
  ↓
git pull origin main
  ↓
docker compose build
  ↓
docker compose down
  ↓
docker compose up -d
  ↓
デプロイ完了 ✅
```

## 5. 動作確認

### 手動で試す場合

```bash
# サーバー上で手動実行して確認
cd ~/workspace/nkmzbot
git pull origin main
docker compose build
docker compose down
docker compose up -d
docker compose ps
```

### GitHub Actionsで確認

1. feat/cicd ブランチを main にマージ
2. GitHubリポジトリの Actions タブを開く
3. "Deploy" ワークフローが実行されることを確認
4. ログを確認して成功を確認

## 6. トラブルシューティング

### SSH接続エラーの場合

```bash
# サーバー側でSSH設定を確認
chmod 700 ~/.ssh
chmod 600 ~/.ssh/authorized_keys
chmod 600 ~/.ssh/github_deploy_key

# SSHデーモンの再起動（必要に応じて）
sudo systemctl restart sshd
```

### Dockerパーミッションエラーの場合

```bash
# ユーザーをdockerグループに追加
sudo usermod -aG docker $USER
# ログアウト・ログインが必要
```

### デプロイパスが違う場合

GitHub SecretsでDEPLOY_PATHを設定：
```
Name: DEPLOY_PATH
Value: /path/to/your/nkmzbot
```

## 7. セキュリティ注意事項

- SSH秘密鍵は**絶対に**コミットしない
- GitHub Secretsに保存した秘密鍵は暗号化されて保存される
- デプロイ用の鍵は専用のものを使用し、他の用途と混在させない
- 可能であれば、SSHポートをデフォルトの22から変更することを推奨

## 8. デプロイの無効化

自動デプロイを一時的に停止したい場合：
- リポジトリの Settings → Actions → Disable workflow を選択
- または `.github/workflows/deploy.yml` を削除
