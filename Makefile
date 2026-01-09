.PHONY: help build test lint fmt clean run docker-build docker-up docker-down migrate-up migrate-down

# デフォルトターゲット
help:
	@echo "Available targets:"
	@echo "  make build         - アプリケーションをビルド"
	@echo "  make test          - テストを実行"
	@echo "  make test-coverage - カバレッジ付きでテストを実行"
	@echo "  make lint          - Lintを実行"
	@echo "  make fmt           - コードをフォーマット"
	@echo "  make clean         - ビルド成果物を削除"
	@echo "  make run           - アプリケーションを実行"
	@echo "  make docker-build  - Dockerイメージをビルド"
	@echo "  make docker-up     - Docker Composeで起動"
	@echo "  make docker-down   - Docker Composeを停止"
	@echo "  make migrate-up    - マイグレーションを実行"
	@echo "  make migrate-down  - マイグレーションをロールバック"

# ビルド
build:
	go build -o nkmzbot cmd/nkmzbot/main.go

# テスト
test:
	go test -v -race ./...

# カバレッジ付きテスト
test-coverage:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "カバレッジレポート: coverage.html"

# Lint
lint:
	@which golangci-lint > /dev/null || test -f $(HOME)/go/bin/golangci-lint || (echo "golangci-lint がインストールされていません。インストールしてください: https://golangci-lint.run/usage/install/" && exit 1)
	@if which golangci-lint > /dev/null 2>&1; then \
		golangci-lint run --timeout=5m; \
	else \
		$(HOME)/go/bin/golangci-lint run --timeout=5m; \
	fi

# フォーマット
fmt:
	go fmt ./...
	goimports -w .

# クリーンアップ
clean:
	rm -f nkmzbot
	rm -f coverage.out coverage.html

# 実行
run:
	go run cmd/nkmzbot/main.go

# Docker
docker-build:
	docker build -t nkmzbot:latest .

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

# マイグレーション（goose が必要）
migrate-up:
	@which goose > /dev/null || (echo "goose がインストールされていません: go install github.com/pressly/goose/v3/cmd/goose@latest" && exit 1)
	goose -dir migrations postgres "$(DATABASE_URL)" up

migrate-down:
	@which goose > /dev/null || (echo "goose がインストールされていません: go install github.com/pressly/goose/v3/cmd/goose@latest" && exit 1)
	goose -dir migrations postgres "$(DATABASE_URL)" down

# 依存関係のインストール
deps:
	go mod download
	go mod tidy

# CI用（全チェック実行）
ci: lint test build
