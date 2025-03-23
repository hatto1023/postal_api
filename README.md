# 郵便番号 API

郵便番号から住所情報を取得するAPI

## 概要

このプロジェクトは以下のことができます：

- 郵便番号から住所を検索
- 東京駅からの距離を計算
- APIの利用回数を記録・表示

## 始め方

### 必要なもの

- Docker と Docker Compose

### 起動方法

```
docker compose up -d
```

## 使い方

### 1. サーバーが起動しているか確認

```
http://localhost:8080
```

「Welcome to Postal API!」と表示されれば成功です。

### 2. 郵便番号から住所を取得

```
http://localhost:8080/address?postal_code=1000001
```

結果の例：
```json
{
  "postal_code": "1000001",
  "hit_count": 1,
  "address": "東京都千代田区千代田",
  "tokyo_sta_distance": 1.3
}
```

### 3. 利用統計を確認

```
http://localhost:8080/address/access_logs
```

結果の例：
```json
{
  "access_logs": [
    {
      "postal_code": "1000001",
      "request_count": 3
    },
    {
      "postal_code": "1020073",
      "request_count": 1
    }
  ]
}
```

## プロジェクトの中身

- `main.go` - APIのプログラム
- `Dockerfile` - Dockerコンテナ作成用
- `docker-compose.yml` - Docker Compose設定
- `init.sql` - データベース初期設定
- `go.mod`, `go.sum` - Goの依存関係ファイル

## APIの詳細

### 住所検索 API

```
/address?postal_code=<7桁の郵便番号>
```

結果の説明：
- `postal_code`: 入力した郵便番号
- `hit_count`: 見つかった住所の数
- `address`: 共通の住所部分
- `tokyo_sta_distance`: 東京駅からの距離（km）

### アクセスログ API

```
/address/access_logs
```

郵便番号ごとの検索回数を表示します（多い順）。
