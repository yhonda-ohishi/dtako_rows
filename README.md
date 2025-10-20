# dtako_rows - Go gRPCサービス

デジタルタコグラフ（DTako）の運行データ（dtako_rows）を管理するgRPCベースのマイクロサービス。

## 概要

`dtako_rows`は、`prod_db`データベースの`dtako_rows`テーブルを管理するgRPCサービスです。[dtako_events](https://github.com/yhonda-ohishi/dtako_events)と同様のアーキテクチャで実装されており、運行データのCRUD操作と検索機能を提供します。

## 機能

- **運行データ管理**
  - GetRow: 運行データ詳細取得
  - ListRows: 運行データ一覧取得（ページング対応）
  - CreateRow: 運行データ作成
  - UpdateRow: 運行データ更新
  - DeleteRow: 運行データ削除
  - SearchRows: 条件検索（日付範囲、車輌CC、乗務員CD1）

## 技術スタック

- **言語**: Go 1.25+
- **プロトコル**: gRPC
- **データベース**: MySQL/MariaDB (prod_db)
- **ORM**: GORM v1.25+
- **アーキテクチャ**: Registryパターン（外部サービス統合対応）

## セットアップ

### 必要要件

- Go 1.25以上
- MySQL/MariaDB (prod_db)
- Protocol Buffers Compiler (protoc)

### インストール

```bash
# リポジトリクローン（またはディレクトリ作成）
cd c:\go\dtako_rows

# 依存関係のインストール
go mod download

# Protocol Buffersのコンパイル
make proto
```

### 環境設定

`.env.example`を`.env`にコピーして編集:

```bash
cp .env.example .env
```

```.env
# データベース設定（prod_db）
DB_HOST=localhost
DB_PORT=3306
DB_USER=your_username
DB_PASSWORD=your_password
DB_NAME=prod_db

# gRPC設定
GRPC_PORT=50053
```

## ビルドと実行

```bash
# ビルド
make build

# 実行
make run

# または直接実行
./bin/dtako_rows.exe
```

## API使用例

### grpcurlを使用した呼び出し

```bash
# GetRow - 運行データ詳細取得
grpcurl -plaintext -d '{
  "id": "202112010001"
}' localhost:50053 dtako_rows.DtakoRowsService/GetRow

# ListRows - 運行データ一覧取得
grpcurl -plaintext -d '{
  "page": 1,
  "page_size": 10,
  "order_by": "出庫日時 DESC"
}' localhost:50053 dtako_rows.DtakoRowsService/ListRows

# SearchRows - 条件検索
grpcurl -plaintext -d '{
  "date_from": "2021-12-01T00:00:00Z",
  "date_to": "2021-12-31T23:59:59Z",
  "sharyou_cc": "1001"
}' localhost:50053 dtako_rows.DtakoRowsService/SearchRows
```

### Goクライアントでの使用例

```go
package main

import (
    "context"
    "log"

    pb "github.com/yhonda-ohishi/dtako_rows/proto"
    "google.golang.org/grpc"
)

func main() {
    conn, err := grpc.Dial("localhost:50053", grpc.WithInsecure())
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()

    client := pb.NewDtakoRowsServiceClient(conn)

    // 運行データ詳細取得
    resp, err := client.GetRow(context.Background(), &pb.GetRowRequest{
        Id: "202112010001",
    })
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("DtakoRow: %+v", resp.Row)
}
```

### 外部サービスから利用（Registryパターン）

```go
package main

import (
    "log"
    "net"

    "github.com/yhonda-ohishi/dtako_rows/pkg/registry"
    "google.golang.org/grpc"
    "gorm.io/gorm"
)

func main() {
    // データベース接続（既存のDB接続を想定）
    var db *gorm.DB
    // db = ... (database connection)

    // gRPCサーバー作成
    grpcServer := grpc.NewServer()

    // dtako_rowsサービスを登録（1行で完了）
    registry.Register(grpcServer, db)

    // その他のサービスも登録可能
    // otherRegistry.Register(grpcServer, db)

    // サーバー起動
    listener, _ := net.Listen("tcp", ":50053")
    grpcServer.Serve(listener)
}
```

## プロジェクト構造

```
dtako_rows/
├── proto/                      # Protocol Buffers定義
│   └── dtako_rows.proto
├── pkg/                        # 公開パッケージ
│   └── registry/              # サービス登録（外部統合用）
│       └── registry.go
├── internal/
│   ├── models/                # GORMモデル
│   │   └── dtako_row.go
│   ├── repository/            # データアクセス層
│   │   └── dtako_row_repository.go
│   ├── service/               # gRPCサービス実装
│   │   └── dtako_rows_service.go
│   └── config/                # 設定管理
│       └── database.go
├── cmd/
│   └── server/
│       └── main.go           # エントリポイント
├── .env.example
├── .gitignore
├── Makefile
├── go.mod
└── README.md
```

## データモデル

### DtakoRow（運行データ）

```go
type DtakoRow struct {
    ID                   string     // 主キー
    UnkoNo               string     // 運行NO
    SharyouCC            string     // 車輌CC
    JomuinCD1            string     // 乗務員CD1
    ShukkoDateTime       time.Time  // 出庫日時
    KikoDateTime         *time.Time // 帰庫日時
    UnkoDate             time.Time  // 運行日
    TaishouJomuinKubun   int        // 対象乗務員区分
    SoukouKyori          float64    // 走行距離
    NenryouShiyou        float64    // 燃料使用量
    Created              time.Time  // 作成日時
    Modified             time.Time  // 更新日時
}
```

## 開発

### Protocol Buffersの再コンパイル

```bash
make proto
```

### テスト実行

```bash
make test

# カバレッジ付き
make test-coverage
```

### クリーンアップ

```bash
make clean
```

## トラブルシューティング

### データベース接続エラー

1. `.env`ファイルの設定を確認
2. MySQLサーバーが起動しているか確認
3. データベース名（prod_db）とユーザー権限を確認

### Protocol Buffersのコンパイルエラー

```bash
# protocのインストール確認
protoc --version

# Go用プラグインのインストール
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

## 参考プロジェクト

- [dtako_events](https://github.com/yhonda-ohishi/dtako_events) - 本プロジェクトの参考実装
- [db_service](https://github.com/yhonda-ohishi/db_service) - リポジトリパターン、サービス自動登録

## ライセンス

MIT License

## 作成者

yhonda-ohishi
