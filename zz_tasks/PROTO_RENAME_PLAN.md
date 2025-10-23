# Proto名変更計画

## 概要

dtako_rowsとdb_serviceのproto名を統一し、明確な命名規則に変更します。

---

## 現状の問題

現在のproto名が統一されていない：

| 場所 | 現在の名前 | 問題点 |
|------|-----------|--------|
| dtako_rowsリポジトリ | `dtako_rows_aggregation.proto` | 長すぎる |
| db_serviceリポジトリ | `dtako_rows.proto` | dtako_rowsと混同しやすい |

---

## 変更計画

### 変更内容

| リポジトリ | 変更前 | 変更後 | 理由 |
|-----------|--------|--------|------|
| dtako_rows | `dtako_rows_aggregation` | `dtako_rows` | シンプルで分かりやすい |
| db_service | `dtako_rows` | `db_dtako_rows` | DB層であることを明示 |

### 変更後の構造

```
buf.build/yhonda-ohishi/dtako-rows
  └─ dtako_rows.proto
       └─ package: dtako_rows
       └─ service: DtakoRowsService

buf.build/yhonda-ohishi/db-service
  └─ db_dtako_rows.proto
       └─ package: db_dtako_rows
       └─ service: DbDtakoRowsService
```

---

## import文の変更

### dtako_rows側（dtako_rows.proto）

```protobuf
syntax = "proto3";
package dtako_rows;
option go_package = "github.com/yhonda-ohishi/dtako_rows/v3/proto;dtako_rows";

import "db_dtako_rows.proto";  // 変更後

service DtakoRowsService {
  rpc GetMonthlyFuelConsumption(GetMonthlyFuelConsumptionRequest) returns (MonthlyFuelConsumptionResponse);
  rpc GetVehicleMonthlySummary(GetVehicleMonthlySummaryRequest) returns (VehicleMonthlySummaryResponse);
  rpc GetDailySummary(GetDailySummaryRequest) returns (DailySummaryResponse);
  rpc ExportMonthlyFuelCSV(GetMonthlyFuelConsumptionRequest) returns (ExportCSVResponse);
}

message GetMonthlyFuelConsumptionRequest {
  string car_cc = 1;
  string start_date = 2;
  string end_date = 3;
}

// db_dtako_rowsのメッセージを使用
// db_dtako_rows.DTakoRows として参照
```

### Go import文の変更

**変更前**:
```go
import (
    dbpb "buf.build/gen/go/yhonda-ohishi/db-service/protocolbuffers/go"
    dbgrpc "buf.build/gen/go/yhonda-ohishi/db-service/grpc/go/_gogrpc"
)

client := dbgrpc.NewDTakoRowsServiceClient(conn)
```

**変更後**:
```go
import (
    dbpb "buf.build/gen/go/yhonda-ohishi/db-service/protocolbuffers/go/db_dtako_rows"
    dbgrpc "buf.build/gen/go/yhonda-ohishi/db-service/grpc/go/db_dtako_rows/_gogrpc"
)

client := dbgrpc.NewDbDtakoRowsServiceClient(conn)
```

---

## 影響範囲

### dtako_rowsリポジトリ

変更が必要なファイル：

1. **proto/dtako_rows_aggregation.proto** → **proto/dtako_rows.proto**（リネーム）
   - package名を変更: `dtako_rows_aggregation` → `dtako_rows`
   - service名を変更: `DtakoRowsAggregationService` → `DtakoRowsService`
   - import文を変更: `"dtako_rows.proto"` → `"db_dtako_rows.proto"`

2. **internal/service/dtako_rows_service.go**
   ```go
   // 変更前
   import (
       dbpb "buf.build/gen/go/yhonda-ohishi/db-service/protocolbuffers/go"
       dbgrpc "buf.build/gen/go/yhonda-ohishi/db-service/grpc/go/_gogrpc"
   )
   type DtakoRowsService struct {
       dbgrpc.UnimplementedDTakoRowsServiceServer
       dbClient dbgrpc.DTakoRowsServiceClient
   }

   // 変更後
   import (
       dbpb "buf.build/gen/go/yhonda-ohishi/db-service/protocolbuffers/go/db_dtako_rows"
       dbgrpc "buf.build/gen/go/yhonda-ohishi/db-service/grpc/go/db_dtako_rows/_gogrpc"
   )
   type DtakoRowsService struct {
       dbgrpc.UnimplementedDbDtakoRowsServiceServer
       dbClient dbgrpc.DbDtakoRowsServiceClient
   }
   ```

3. **internal/service/aggregation_service.go**
   ```go
   // 変更前
   import (
       pb "github.com/yhonda-ohishi/dtako_rows/v3/proto"
       dbgrpc "buf.build/gen/go/yhonda-ohishi/db-service/grpc/go/_gogrpc"
   )
   type DtakoRowsAggregationService struct {
       pb.UnimplementedDtakoRowsAggregationServiceServer
       dbClient dbgrpc.DTakoRowsServiceClient
   }

   // 変更後
   import (
       pb "github.com/yhonda-ohishi/dtako_rows/v3/proto"
       dbgrpc "buf.build/gen/go/yhonda-ohishi/db-service/grpc/go/db_dtako_rows/_gogrpc"
   )
   type DtakoRowsService struct {
       pb.UnimplementedDtakoRowsServiceServer
       dbClient dbgrpc.DbDtakoRowsServiceClient
   }
   ```

4. **cmd/server/main.go**
   ```go
   // 変更前
   dbgrpc "buf.build/gen/go/yhonda-ohishi/db-service/grpc/go/_gogrpc"
   pb "github.com/yhonda-ohishi/dtako_rows/v3/proto"

   dtakoRowsClient := dbgrpc.NewDTakoRowsServiceClient(conn)
   dbgrpc.RegisterDTakoRowsServiceServer(grpcServer, dtakoRowsService)
   pb.RegisterDtakoRowsAggregationServiceServer(grpcServer, aggregationService)

   // 変更後
   dbgrpc "buf.build/gen/go/yhonda-ohishi/db-service/grpc/go/db_dtako_rows/_gogrpc"
   pb "github.com/yhonda-ohishi/dtako_rows/v3/proto"

   dtakoRowsClient := dbgrpc.NewDbDtakoRowsServiceClient(conn)
   dbgrpc.RegisterDbDtakoRowsServiceServer(grpcServer, dtakoRowsService)
   pb.RegisterDtakoRowsServiceServer(grpcServer, aggregationService)
   ```

5. **pkg/registry/registry.go**
   ```go
   // 変更前
   dbgrpc "buf.build/gen/go/yhonda-ohishi/db-service/grpc/go/_gogrpc"
   pb "github.com/yhonda-ohishi/dtako_rows/v3/proto"

   func RegisterWithClient(grpcServer *grpc.Server, dbClient dbgrpc.DTakoRowsServiceClient) {
       dbgrpc.RegisterDTakoRowsServiceServer(grpcServer, svc)
       pb.RegisterDtakoRowsAggregationServiceServer(grpcServer, aggSvc)
   }

   // 変更後
   dbgrpc "buf.build/gen/go/yhonda-ohishi/db-service/grpc/go/db_dtako_rows/_gogrpc"
   pb "github.com/yhonda-ohishi/dtako_rows/v3/proto"

   func RegisterWithClient(grpcServer *grpc.Server, dbClient dbgrpc.DbDtakoRowsServiceClient) {
       dbgrpc.RegisterDbDtakoRowsServiceServer(grpcServer, svc)
       pb.RegisterDtakoRowsServiceServer(grpcServer, aggSvc)
   }
   ```

6. **go.mod**
   ```go
   // 変更前
   require (
       buf.build/gen/go/yhonda-ohishi/db-service/grpc/go v1.5.1-20251022140655-2e935c1145cc.2
       buf.build/gen/go/yhonda-ohishi/db-service/protocolbuffers/go v1.36.10-20251022140655-2e935c1145cc.1
   )

   // 変更後（新しいバージョンに更新）
   require (
       buf.build/gen/go/yhonda-ohishi/db-service/grpc/go v1.5.1-XXXXXXXX.2
       buf.build/gen/go/yhonda-ohishi/db-service/protocolbuffers/go v1.36.10-XXXXXXXX.1
   )
   ```

### db_serviceリポジトリ（desktop-server内）

変更が必要なファイル：

1. **proto/dtako_rows.proto** → **proto/db_dtako_rows.proto**（リネーム）
   ```protobuf
   // 変更前
   syntax = "proto3";
   package dtako_rows;
   option go_package = "github.com/yhonda-ohishi/desktop-server/proto;dtako_rows";

   service DTakoRowsService {
     rpc Get(GetDTakoRowsRequest) returns (DTakoRowsResponse);
     rpc List(ListDTakoRowsRequest) returns (ListDTakoRowsResponse);
     rpc GetByOperationNo(GetDTakoRowsByOperationNoRequest) returns (ListDTakoRowsResponse);
   }

   // 変更後
   syntax = "proto3";
   package db_dtako_rows;
   option go_package = "github.com/yhonda-ohishi/desktop-server/proto;db_dtako_rows";

   service DbDtakoRowsService {
     rpc Get(GetDTakoRowsRequest) returns (DTakoRowsResponse);
     rpc List(ListDTakoRowsRequest) returns (ListDTakoRowsResponse);
     rpc GetByOperationNo(GetDTakoRowsByOperationNoRequest) returns (ListDTakoRowsResponse);
   }
   ```

2. **internal/service/db_dtako_rows_service.go**（ファイル名も変更）
   - package名、import文、サービス構造体名を全て変更

3. **cmd/server/main.go**
   - import文とサービス登録を更新

4. **buf.yaml**
   ```yaml
   # 変更不要（module名は変わらない）
   version: v2
   modules:
     - path: proto
       name: buf.build/yhonda-ohishi/db-service
   ```

### desktop-serverリポジトリ（統合側）

1. **main.go**
   ```go
   // 変更前
   import (
       dtako_rows_registry "github.com/yhonda-ohishi/dtako_rows/v3/pkg/registry"
       dbgrpc "buf.build/gen/go/yhonda-ohishi/db-service/grpc/go/_gogrpc"
   )

   dtakoRowsClient := dbgrpc.NewDTakoRowsServiceClient(localConn)
   dtako_rows_registry.RegisterWithClient(grpcServer, dtakoRowsClient)

   // 変更後
   import (
       dtako_rows_registry "github.com/yhonda-ohishi/dtako_rows/v3/pkg/registry"
       dbgrpc "buf.build/gen/go/yhonda-ohishi/db-service/grpc/go/db_dtako_rows/_gogrpc"
   )

   dtakoRowsClient := dbgrpc.NewDbDtakoRowsServiceClient(localConn)
   dtako_rows_registry.RegisterWithClient(grpcServer, dtakoRowsClient)
   ```

---

## 移行手順

### ステップ1: db_service側を先に変更

```bash
cd C:/go/desktop-server

# 1. protoファイルをリネーム
git mv proto/dtako_rows.proto proto/db_dtako_rows.proto

# 2. proto内容を編集
# - package名: dtako_rows → db_dtako_rows
# - service名: DTakoRowsService → DbDtakoRowsService
# - go_package: ;dtako_rows → ;db_dtako_rows

# 3. コード生成
buf generate

# 4. 実装ファイル全てを更新
# - import文の更新
# - サービス構造体名の更新
# - メソッド実装の更新

# 5. ビルド確認
go build ./...

# 6. コミット & BSRにプッシュ
git add .
git commit -m "Proto名変更: dtako_rows → db_dtako_rows

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"

git push origin master
git tag v2.0.0
git push origin v2.0.0

buf push --tag v2.0.0
```

### ステップ2: dtako_rows側を更新

```bash
cd C:/go/dtako_rows

# 1. buf.yamlのdepsを最新バージョンに更新
# deps:
#   - buf.build/yhonda-ohishi/db-service  # 最新のv2.0.0を参照

# 2. Go依存関係を更新
go get buf.build/gen/go/yhonda-ohishi/db-service/grpc/go@latest
go get buf.build/gen/go/yhonda-ohishi/db-service/protocolbuffers/go@latest
go mod tidy

# 3. protoファイルをリネーム
git mv proto/dtako_rows_aggregation.proto proto/dtako_rows.proto

# 4. proto内容を編集
# - package名: dtako_rows_aggregation → dtako_rows
# - service名: DtakoRowsAggregationService → DtakoRowsService
# - import: "dtako_rows.proto" → "db_dtako_rows.proto"

# 5. コード生成
buf generate

# 6. 実装ファイル全てを更新
# - internal/service/dtako_rows_service.go
# - internal/service/aggregation_service.go
# - cmd/server/main.go
# - pkg/registry/registry.go

# 7. ビルド確認
go build ./...

# 8. コミット & BSRにプッシュ
git add .
git commit -m "Proto名変更: dtako_rows_aggregation → dtako_rows

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"

git push origin master
git tag v2.0.0
git push origin v2.0.0

buf push --tag v2.0.0
```

### ステップ3: desktop-server統合を更新

```bash
cd C:/go/desktop-server

# 1. dtako_rowsの新しいバージョンに更新
go get github.com/yhonda-ohishi/dtako_rows/v3@v2.0.0
go mod tidy

# 2. import文とサービス登録を更新
# - main.goのimport文
# - サービスクライアント作成
# - レジストリ呼び出し

# 3. ビルド確認
go build ./...

# 4. テスト実行
go run test_aggregation.go  # または該当するテスト

# 5. コミット & プッシュ
git add .
git commit -m "dtako_rows v2.0.0に更新（proto名変更対応）

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"

git push origin master
```

---

## チェックリスト

### db_service側

- [ ] `proto/dtako_rows.proto` → `proto/db_dtako_rows.proto` リネーム
- [ ] proto内のpackage名を `db_dtako_rows` に変更
- [ ] proto内のservice名を `DbDtakoRowsService` に変更
- [ ] proto内のgo_packageを `;db_dtako_rows` に変更
- [ ] `buf generate` 実行
- [ ] サービス実装ファイルのimport文を更新
- [ ] サービス構造体名を更新
- [ ] `go build ./...` 成功確認
- [ ] git commit & push
- [ ] git tag v2.0.0 & push
- [ ] `buf push --tag v2.0.0` 実行

### dtako_rows側

- [ ] buf.yamlのdepsを更新
- [ ] `go get` で最新のdb-service依存関係を取得
- [ ] `proto/dtako_rows_aggregation.proto` → `proto/dtako_rows.proto` リネーム
- [ ] proto内のpackage名を `dtako_rows` に変更
- [ ] proto内のservice名を `DtakoRowsService` に変更
- [ ] proto内のimportを `"db_dtako_rows.proto"` に変更
- [ ] `buf generate` 実行
- [ ] `internal/service/dtako_rows_service.go` 更新
- [ ] `internal/service/aggregation_service.go` 更新
- [ ] `cmd/server/main.go` 更新
- [ ] `pkg/registry/registry.go` 更新
- [ ] `go build ./...` 成功確認
- [ ] git commit & push
- [ ] git tag v2.0.0 & push
- [ ] `buf push --tag v2.0.0` 実行

### desktop-server統合側

- [ ] `go get github.com/yhonda-ohishi/dtako_rows/v3@v2.0.0`
- [ ] main.goのimport文を更新
- [ ] サービスクライアント作成を更新
- [ ] レジストリ呼び出しを更新
- [ ] `go build ./...` 成功確認
- [ ] テスト実行して動作確認
- [ ] git commit & push

---

## メリット

1. **明確な命名**: プロト名からどのリポジトリか一目瞭然
   - `dtako_rows`: ビジネスロジック層
   - `db_dtako_rows`: データアクセス層

2. **衝突回避**: 同じ名前のprotoが存在しないため混乱がない

3. **保守性向上**: 将来の拡張時にどこに何を追加すべきか明確

4. **一貫性**: サービス名とpackage名が統一される

---

## 注意事項

### バージョン管理

- BSRの新しいバージョン（v2.0.0）として公開
- 既存のv1.xユーザーは影響を受けない（後方互換性なし）
- 移行期間中はv1.xとv2.xが併存可能

### 破壊的変更

この変更は**破壊的変更**です：

- package名が変更されるため、既存のimportが動作しなくなる
- service名が変更されるため、gRPCクライアントの作成コードが変更必要
- BSRのメジャーバージョンを上げる（v1.x → v2.0.0）

### 移行期間

1. v1.xは引き続き使用可能（BSRに残る）
2. 新規プロジェクトはv2.0.0を使用
3. 既存プロジェクトは段階的にv2.0.0に移行

---

## トラブルシューティング

### エラー: package not found

```
cannot find package "buf.build/gen/go/yhonda-ohishi/db-service/protocolbuffers/go"
```

**原因**: 古いimport pathを使用している

**対処法**:
```go
// 修正前
import dbpb "buf.build/gen/go/yhonda-ohishi/db-service/protocolbuffers/go"

// 修正後
import dbpb "buf.build/gen/go/yhonda-ohishi/db-service/protocolbuffers/go/db_dtako_rows"
```

### エラー: service name mismatch

```
undefined: dbgrpc.NewDTakoRowsServiceClient
```

**原因**: サービス名が変更されている

**対処法**:
```go
// 修正前
client := dbgrpc.NewDTakoRowsServiceClient(conn)

// 修正後
client := dbgrpc.NewDbDtakoRowsServiceClient(conn)
```

### エラー: buf push failed

```
validation failed: module has uncommitted changes
```

**原因**: gitの変更がコミットされていない

**対処法**:
```bash
git add .
git commit -m "Proto name change"
buf push --tag v2.0.0
```

---

## 完了後の確認

1. **BSR確認**: https://buf.build/yhonda-ohishi/dtako-rows でv2.0.0が公開されているか
2. **BSR確認**: https://buf.build/yhonda-ohishi/db-service でv2.0.0が公開されているか
3. **ビルド確認**: 全リポジトリで `go build ./...` が成功するか
4. **テスト確認**: 統合テストが正常に動作するか
5. **ドキュメント更新**: READMEやSPECIFICATION.mdを更新

---

## 参考リンク

- [Buf Schema Registry](https://buf.build/)
- [gRPC Service Definition](https://grpc.io/docs/what-is-grpc/core-concepts/)
- [Semantic Versioning](https://semver.org/)
