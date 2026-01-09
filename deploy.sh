#!/bin/bash
# デプロイスクリプト - ローカルテスト用

set -e  # エラーが発生したら即座に終了

echo "📦 Pulling latest code..."
git pull origin main

echo "🔨 Rebuilding Docker image..."
docker compose build

echo "🔄 Restarting bot..."
docker compose down
docker compose up -d

echo "✅ Deployment completed!"
docker compose ps

echo ""
echo "📋 Recent logs:"
docker compose logs --tail=20
