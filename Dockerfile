# ビルドステージ
FROM golang:1.24-alpine AS builder

# 作業ディレクトリの設定
WORKDIR /app

# Go モジュールの依存関係をコピー
COPY go.mod go.sum ./

# 依存関係のダウンロード
RUN go mod download

# ソースコードをコピー
COPY . .

# アプリケーションのビルド
RUN go build -o postal-api .

# 軽量な本番イメージ
FROM alpine:latest

# 必要なツールのインストール
RUN apk --no-cache add ca-certificates

# 作業ディレクトリの設定
WORKDIR /root/

# ビルドステージからバイナリをコピー
COPY --from=builder /app/postal-api .

# アプリケーションが使用するポートを公開
EXPOSE 8080

# アプリケーションの実行
CMD ["./postal-api"]