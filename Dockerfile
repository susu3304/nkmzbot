# Rust公式イメージを使用
FROM rust:latest

# 作業ディレクトリを作成
WORKDIR /app

# ソースコードをコピー
COPY . .

# 依存関係をビルド
RUN cargo build --release

# 実行ファイルを起動
CMD ["./target/release/nkmzbot"]