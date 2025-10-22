# Rust公式slimイメージを使用
FROM rust:slim

# 作業ディレクトリを作成
WORKDIR /app

# ソースコードをコピー
COPY . .

# 必要なシステム依存関係をインストール（pkg-config と OpenSSL の開発ヘッダ）
# Debian/Ubuntu 系の slim イメージを想定しているため apt を使う
RUN apt-get update \
	&& DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
	   pkg-config \
	   libssl-dev \
	   build-essential \
	   ca-certificates \
	&& rm -rf /var/lib/apt/lists/*

# 依存関係をビルド
RUN cargo build --release

# 実行ファイルを起動
CMD ["./target/release/nkmzbot"]