# dtako_rows サービス仕様書

## 概要

運行データ（DTakoRows）の集計・分析機能を提供するgRPCサービス。
db_serviceから取得したデータに対してビジネスロジックを適用し、月次・日次の給油量集計などを行う。

---

## アーキテクチャ

```
フロントエンド
  ↓ gRPC
desktop-server (localhost:50051)
  ├─ db_service (DBアクセス層)
  └─ dtako_rows (ビジネスロジック層)
       ↓ 内部gRPC呼び出し
       db_service
```

### サービス構成

| サービス | 説明 | ポート |
|---------|------|--------|
| db_service.DTakoRowsService | DB直接アクセス（CRUD） | 50051 |
| dtako_rows_aggregation.DtakoRowsAggregationService | 集計ロジック | 50053 (スタンドアロン時) |

---

## Proto定義

### パッケージ

```
buf.build/yhonda-ohishi/dtako-rows
  └─ dtako_rows_aggregation.proto
```

### 依存関係

```yaml
deps:
  - buf.build/yhonda-ohishi/db-service
```

---

## API仕様

### 1. GetMonthlyFuelConsumption

**特定車両の月次給油量集計**

#### Request
```protobuf
message GetMonthlyFuelConsumptionRequest {
  string car_cc = 1;      // 車輌CC（必須）
  string start_date = 2;  // 開始日 (YYYY-MM-DD)
  string end_date = 3;    // 終了日 (YYYY-MM-DD)
}
```

#### Response
```protobuf
message MonthlyFuelConsumptionResponse {
  repeated MonthlyFuelSummary summaries = 1;
  string car_cc = 2;
  string period = 3;  // 集計期間
}

message MonthlyFuelSummary {
  string car_cc = 1;
  string year_month = 2;       // YYYY-MM
  double total_distance = 3;   // km
  double total_fuel = 4;       // L
  int32 trip_count = 5;
  double avg_fuel_efficiency = 6; // km/L
}
```

#### 使用例
```typescript
const response = await client.getMonthlyFuelConsumption({
  carCc: "215800",
  startDate: "2025-09-01",
  endDate: "2025-10-31"
});
```

#### 処理内容
1. db_serviceから指定期間の全運行データを取得
2. 指定車両でフィルタリング
3. 月ごとに集計
   - 走行距離の合計
   - 運行回数のカウント
   - 給油量の推定計算（距離 ÷ 燃費）

#### パフォーマンス
- **現在**: 約22秒/1000件
- **改善案（将来）**: ストリーミングRPCで5秒以下

---

### 2. GetVehicleMonthlySummary

**全車両の月次サマリー取得**

#### Request
```protobuf
message GetVehicleMonthlySummaryRequest {
  string start_date = 1;
  string end_date = 2;
}
```

#### Response
```protobuf
message VehicleMonthlySummaryResponse {
  repeated VehicleMonthlySummaries vehicle_summaries = 1;
  int32 total_vehicles = 2;
  string period = 3;
}

message VehicleMonthlySummaries {
  string car_cc = 1;
  repeated MonthlyFuelSummary summaries = 2;
}
```

#### 使用例
```typescript
const response = await client.getVehicleMonthlySummary({
  startDate: "2025-09-01",
  endDate: "2025-10-31"
});

console.log(`総車両数: ${response.totalVehicles}`);
response.vehicleSummaries.forEach(v => {
  console.log(`車両: ${v.carCc}`);
  v.summaries.forEach(s => {
    console.log(`  ${s.yearMonth}: ${s.totalDistance}km`);
  });
});
```

---

### 3. GetDailySummary

**日次サマリー取得**

#### Request
```protobuf
message GetDailySummaryRequest {
  string car_cc = 1;
  string start_date = 2;
  string end_date = 3;
}
```

#### Response
```protobuf
message DailySummaryResponse {
  repeated DailySummary summaries = 1;
  string car_cc = 2;
  string period = 3;
}

message DailySummary {
  string car_cc = 1;
  string date = 2;            // YYYY-MM-DD
  double total_distance = 3;
  double total_fuel = 4;
  int32 trip_count = 5;
}
```

---

### 4. ExportMonthlyFuelCSV

**CSV形式でエクスポート**

#### Request
```protobuf
message GetMonthlyFuelConsumptionRequest {
  // GetMonthlyFuelConsumptionと同じ
}
```

#### Response
```protobuf
message ExportCSVResponse {
  string csv_data = 1;
  string filename = 2;
}
```

#### CSV形式
```csv
年月,車両CC,走行距離(km),給油量(L),運行回数,平均燃費(km/L)
2025-10,215800,8845.9,884.6,3,10.00
```

---

## ビジネスロジック

### 給油量の計算

実際の給油データがない場合、以下の式で推定：

```
給油量 (L) = 走行距離 (km) ÷ 燃費 (km/L)
```

**デフォルト燃費**: 10.0 km/L

**TODO**: 車両マスタから実際の燃費を取得

### フィルタリング

- 車両CC完全一致
- 運行日の範囲チェック（RFC3339形式でパース）
- 走行距離0のデータは除外（オプション）

### デフォルト値

- `List`メソッド:
  - limit: 100（最大1000）
  - order_by: "read_date DESC"

---

## デプロイ構成

### スタンドアロン実行

```bash
# 環境変数
DB_SERVICE_ADDR=localhost:50051
GRPC_PORT=50053

# 起動
./bin/server.exe
```

登録されるサービス:
- `db_service.DTakoRowsService` (プロキシ)
- `dtako_rows_aggregation.DtakoRowsAggregationService` (集計)

### desktop-server統合

```go
import (
    dbgrpc "buf.build/gen/go/yhonda-ohishi/db-service/grpc/go/_gogrpc"
    dtako_rows_registry "github.com/yhonda-ohishi/dtako_rows/v3/pkg/registry"
)

// 同一プロセス内接続
localConn := ... // 内部gRPC接続

// db_serviceクライアント作成
dtakoRowsClient := dbgrpc.NewDTakoRowsServiceClient(localConn)

// 両方のサービスを登録
dtako_rows_registry.RegisterWithClient(grpcServer, dtakoRowsClient)
```

---

## パフォーマンス

### 現在の性能

| 操作 | データ量 | 処理時間 |
|-----|---------|---------|
| GetVehicleMonthlySummary | 940件 | 22秒 |
| GetMonthlyFuelConsumption | 3件（フィルタ後） | 22秒 |
| GetDailySummary | 0件 | 23秒 |

### ボトルネック

1. **gRPC通信**: 940件取得に22秒
2. **全件スキャン**: db_serviceにフィルタ機能がない
3. **重複取得**: 各メソッドで同じデータを再取得

### 改善計画

#### Phase 1: キャッシング（短期）
- メモリ内キャッシュで重複取得を削減
- 期待効果: 67秒 → 22秒

#### Phase 2: db_serviceフィルタ拡張（中期）
```protobuf
message ListDTakoRowsRequest {
  optional string car_cc = 4;
  optional string start_date = 5;
  optional string end_date = 6;
}
```
- 期待効果: 22秒 → 0.1秒

#### Phase 3: ストリーミングRPC（長期）
```protobuf
rpc ListStream(ListDTakoRowsRequest) returns (stream DTakoRows);
```
- 期待効果: 22秒 → 5秒以下
- メリット:
  - メモリ効率向上
  - 処理の並行化
  - 早期レスポンス

---

## エラーハンドリング

### バリデーションエラー

```
InvalidArgument: car_cc is required
InvalidArgument: operation_no is required
```

### データ不在

```
NotFound: row not found
```

### 接続エラー

```
Unavailable: db_service connection failed
```

---

## テスト

### 動作確認済み

```bash
# テストプログラム実行
go run test_aggregation.go

# 結果
Total vehicles: 130
First Vehicle: 215800
  2025-10: 距離=8845.9km, 給油=884.6L, 運行=3回
```

### 推奨テストケース

1. **正常系**
   - 1車両、1ヶ月のデータ取得
   - 全車両のサマリー取得
   - CSV出力

2. **異常系**
   - 存在しない車両CC
   - 不正な日付形式
   - データが0件の期間

3. **パフォーマンス**
   - 1000件以上のデータ処理
   - 複数車両の並行リクエスト

---

## 今後の拡張

### 機能追加案

1. **統計機能**
   - 平均燃費ランキング
   - 走行距離トップ10
   - 月次比較（前月比）

2. **グラフデータ**
   - 時系列データ（Chart.js用）
   - 円グラフデータ（車両別割合）

3. **アラート**
   - 燃費異常検知
   - 走行距離しきい値超過通知

### Proto拡張

```protobuf
// 統計サービス
service DtakoRowsStatisticsService {
  rpc GetFuelEfficiencyRanking(...) returns (...);
  rpc GetAnomalyDetection(...) returns (...);
}
```

---

## バージョン管理

### BSR公開

- **リポジトリ**: `buf.build/yhonda-ohishi/dtako-rows`
- **最新コミット**: `9b318ce64abe4ad28a40c193425146e5`
- **可視性**: public

### セマンティックバージョニング

- **v1.x**: 現在の実装
- **v2.x**: ストリーミングRPC対応（予定）
- **v3.x**: 統計機能追加（予定）

### 互換性ポリシー

- マイナーバージョン: 後方互換性あり（フィールド追加のみ）
- メジャーバージョン: 破壊的変更OK（フィールド削除・型変更）

---

## 参考リンク

- [Buf Schema Registry](https://buf.build/yhonda-ohishi/dtako-rows)
- [db_service Proto](https://buf.build/yhonda-ohishi/db-service)
- [gRPC Streaming Guide](https://grpc.io/docs/what-is-grpc/core-concepts/#server-streaming-rpc)

---

## サービス層フィルタリング機能

### 概要

db_serviceにフィルタ機能がない場合でも、dtako_rowsサービス層で柔軟なフィルタリングが可能です。

### FilterOptions 構造体

```go
type FilterOptions struct {
    CarCC              *string    // 車輌CC（完全一致）
    StartDate          *time.Time // 運行開始日（以降）
    EndDate            *time.Time // 運行終了日（以前）
    MinDistance        *float64   // 最小走行距離
    OperationNos       []string   // 運行NO（複数指定可）
    ExcludeZeroDistance bool      // 走行距離0のデータを除外
}
```

### 主要メソッド

#### ListWithFilter

フィルタオプション付きでデータを取得します。

```go
func (s *DtakoRowsService) ListWithFilter(
    ctx context.Context,
    filter *FilterOptions,
    limit int32,
    offset int32,
) ([]*dbpb.DTakoRows, int32, error)
```

**処理フロー**:
1. db_serviceから1000件ずつページネーションで取得
2. 各行に対してフィルタ条件をチェック
3. 条件に一致するデータのみ収集
4. 指定されたlimit/offsetでページネーション処理
5. フィルタ後の総件数とデータを返却

**最適化**:
- 必要な件数が集まったら早期終了
- フィルタ条件がnullの場合は全データを返却

#### ヘルパーメソッド

よく使うフィルタパターンを簡単に使えるヘルパーメソッド：

```go
// 車輌CCで絞り込み
func (s *DtakoRowsService) ListByCarCC(
    ctx context.Context,
    carCC string,
    limit int32,
) ([]*dbpb.DTakoRows, error)

// 日付範囲で絞り込み
func (s *DtakoRowsService) ListByDateRange(
    ctx context.Context,
    startDate, endDate string,
    limit int32,
) ([]*dbpb.DTakoRows, error)

// 車輌CC + 日付範囲で絞り込み
func (s *DtakoRowsService) ListByCarCCAndDateRange(
    ctx context.Context,
    carCC, startDate, endDate string,
    limit int32,
) ([]*dbpb.DTakoRows, error)
```

### 使用例

#### 基本的な使い方

```go
// 特定車両の最近1ヶ月のデータ
rows, err := service.ListByCarCCAndDateRange(
    ctx,
    "215800",
    "2025-09-01",
    "2025-10-31",
    100, // limit
)
```

#### 複雑なフィルタ

```go
// カスタムフィルタ
minDist := 10.0
filter := &FilterOptions{
    CarCC:              &carCC,
    StartDate:          &startDate,
    EndDate:            &endDate,
    MinDistance:        &minDist,
    ExcludeZeroDistance: true,
}

rows, totalCount, err := service.ListWithFilter(ctx, filter, 100, 0)
```

#### 複数運行NOで絞り込み

```go
filter := &FilterOptions{
    OperationNos: []string{"OP001", "OP002", "OP003"},
}

rows, _, err := service.ListWithFilter(ctx, filter, 0, 0)
```

### 集計メソッドでの活用

集計メソッドは内部でヘルパーメソッドを使用しています：

```go
// GetMonthlyFuelConsumption内部
allRows, err := s.ListByCarCCAndDateRange(ctx, carCC, startDate, endDate, 0)
// → フィルタリング済みデータで集計処理

// GetVehicleMonthlySummary内部
allRows, err := s.ListByDateRange(ctx, startDate, endDate, 0)
// → 期間でフィルタしてから車両別に集計

// GetDailySummary内部
allRows, err := s.ListByCarCCAndDateRange(ctx, carCC, startDate, endDate, 0)
// → フィルタリング済みデータで日次集計
```

### パフォーマンス特性

#### メリット

- **柔軟性**: db_serviceを変更せずに複雑なフィルタが可能
- **保守性**: フィルタロジックがサービス層に集約
- **再利用性**: 複数の集計メソッドで共通のフィルタを使用

#### デメリット

- **パフォーマンス**: 全件取得してからフィルタリング（現状22秒/1000件）
- **メモリ**: 一時的に全データをメモリに保持

#### 改善案

将来的にはdb_service側でフィルタリングを実装することで大幅な性能向上が期待できます：

```protobuf
message ListDTakoRowsRequest {
  int32 limit = 1;
  int32 offset = 2;
  optional string order_by = 3;

  // フィルタ追加（Phase 2）
  optional string car_cc = 4;
  optional string start_date = 5;
  optional string end_date = 6;
  optional double min_distance = 7;
}
```

期待効果: 22秒 → 0.1秒（DB側でWHERE句によるフィルタリング）

---

## Proto名変更計画

### 現状の問題

現在のproto名が統一されていない：

| 場所 | 現在の名前 | 問題点 |
|------|-----------|--------|
| dtako_rowsリポジトリ | `dtako_rows_aggregation.proto` | 長すぎる |
| db_serviceリポジトリ | `dtako_rows.proto` | dtako_rowsと混同しやすい |

### 変更計画

#### 変更内容

| リポジトリ | 変更前 | 変更後 | 理由 |
|-----------|--------|--------|------|
| dtako_rows | `dtako_rows_aggregation` | `dtako_rows` | シンプルで分かりやすい |
| db_service | `dtako_rows` | `db_dtako_rows` | DB層であることを明示 |

#### 変更後の構造

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

#### import文の変更

**dtako_rows側（dtako_rows.proto）**:
```protobuf
syntax = "proto3";
package dtako_rows;

import "db_dtako_rows.proto";  // 変更後

// メッセージ定義でdb_dtako_rowsを参照
message GetMonthlyFuelConsumptionRequest {
  string car_cc = 1;
  string start_date = 2;
  string end_date = 3;
}

// db_dtako_rowsのメッセージを使用
// db_dtako_rows.DTakoRows として参照
```

#### Go import文の変更

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

#### 影響範囲

変更が必要なファイル：

**dtako_rowsリポジトリ**:
1. `proto/dtako_rows_aggregation.proto` → `proto/dtako_rows.proto`（リネーム）
2. `internal/service/dtako_rows_service.go` - import文とサービス名
3. `internal/service/aggregation_service.go` - import文
4. `cmd/server/main.go` - import文とサービス登録
5. `pkg/registry/registry.go` - import文とサービス登録
6. `go.mod` - BSR依存関係の更新

**db_serviceリポジトリ**:
1. `proto/dtako_rows.proto` → `proto/db_dtako_rows.proto`（リネーム）
2. サービス実装ファイル全て - import文とサービス名
3. `buf.yaml` - module設定
4. `buf push`で新しいバージョンをBSRに公開

**desktop-serverリポジトリ**:
1. import文の更新
2. サービス登録コードの更新

#### 移行手順

1. **db_service側を先に変更**:
   ```bash
   cd C:/go/desktop-server
   git mv proto/dtako_rows.proto proto/db_dtako_rows.proto
   # proto内容を編集（package名、service名）
   buf generate
   # 実装ファイル全てを更新
   buf push --tag v2.0.0
   ```

2. **dtako_rows側を更新**:
   ```bash
   cd C:/go/dtako_rows
   # buf.yamlのdepsを最新バージョンに更新
   go get buf.build/gen/go/yhonda-ohishi/db-service/...@latest
   git mv proto/dtako_rows_aggregation.proto proto/dtako_rows.proto
   # proto内容を編集
   buf generate
   # 実装ファイル全てを更新
   buf push --tag v2.0.0
   ```

3. **desktop-server統合を更新**:
   ```bash
   cd C:/go/desktop-server
   # 新しいバージョンに更新
   go get github.com/yhonda-ohishi/dtako_rows/v3@latest
   # import文とサービス登録を更新
   ```

#### メリット

1. **明確な命名**: プロト名からどのリポジトリか一目瞭然
2. **衝突回避**: `dtako_rows`と`db_dtako_rows`で区別が明確
3. **保守性向上**: 将来の拡張時に混乱が少ない

#### 注意事項

- BSRの新しいバージョン（v2.0.0）として公開
- 既存のv1.xユーザーは影響を受けない（後方互換性なし）
- 移行期間中はv1.xとv2.xが併存

---

## 更新履歴

| 日付 | バージョン | 変更内容 |
|-----|-----------|---------|
| 2025-10-23 | v1.0.0 | 初版作成、集計サービス実装 |
| 2025-10-23 | v1.1.0 | サービス層フィルタリング機能追加 |
| TBD | v2.0.0 | Proto名変更（dtako_rows/db_dtako_rows）、ストリーミングRPC対応（予定） |
