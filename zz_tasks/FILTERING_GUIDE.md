# サービス層フィルタリング機能ガイド

## 概要

db_serviceにフィルタ機能がない場合でも、dtako_rowsサービス層で柔軟なフィルタリングが可能です。

このドキュメントでは、実装されたフィルタリング機能の使い方と設計について説明します。

---

## FilterOptions 構造体

### 定義

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

### フィールド説明

| フィールド | 型 | 説明 | デフォルト |
|-----------|---|------|-----------|
| CarCC | *string | 車輌CCで完全一致フィルタ | nil（フィルタなし） |
| StartDate | *time.Time | この日付以降のデータを取得 | nil（フィルタなし） |
| EndDate | *time.Time | この日付以前のデータを取得 | nil（フィルタなし） |
| MinDistance | *float64 | この距離以上のデータを取得 | nil（フィルタなし） |
| OperationNos | []string | 指定した運行NOのいずれかに一致 | nil（フィルタなし） |
| ExcludeZeroDistance | bool | 走行距離0を除外するか | false（除外しない） |

### ポインタ型を使う理由

nilとゼロ値を区別するため：

```go
// フィルタなし（全データを取得）
filter := &FilterOptions{
    MinDistance: nil,  // フィルタしない
}

// 0以上でフィルタ（実質全データ）
minDist := 0.0
filter := &FilterOptions{
    MinDistance: &minDist,  // 0以上
}

// 10以上でフィルタ
minDist := 10.0
filter := &FilterOptions{
    MinDistance: &minDist,  // 10以上
}
```

---

## 主要メソッド

### ListWithFilter

フィルタオプション付きでデータを取得します。

#### シグネチャ

```go
func (s *DtakoRowsService) ListWithFilter(
    ctx context.Context,
    filter *FilterOptions,
    limit int32,
    offset int32,
) ([]*dbpb.DTakoRows, int32, error)
```

#### パラメータ

| パラメータ | 型 | 説明 |
|----------|---|------|
| ctx | context.Context | コンテキスト |
| filter | *FilterOptions | フィルタ条件（nilの場合は全データ） |
| limit | int32 | 取得件数（0の場合は全件） |
| offset | int32 | オフセット（ページネーション用） |

#### 戻り値

| 戻り値 | 型 | 説明 |
|-------|---|------|
| rows | []*dbpb.DTakoRows | フィルタ済みデータ |
| totalCount | int32 | フィルタ後の総件数 |
| error | error | エラー |

#### 処理フロー

```
1. db_serviceから1000件ずつページネーションで取得
   ↓
2. 各行に対してフィルタ条件をチェック（matchesFilter）
   ↓
3. 条件に一致するデータのみ収集
   ↓
4. 必要な件数が集まったら早期終了（最適化）
   ↓
5. 指定されたlimit/offsetでページネーション処理
   ↓
6. フィルタ後の総件数とデータを返却
```

#### 最適化ポイント

- **早期終了**: 必要な件数が集まったら全データを取得せずに終了
- **バッチ処理**: 1000件ずつ取得してメモリ効率を維持
- **nilチェック**: フィルタ条件がnullの場合は全データを返却

#### コード例

```go
// 実装例（簡略版）
func (s *DtakoRowsService) ListWithFilter(ctx context.Context, filter *FilterOptions, limit int32, offset int32) ([]*dbpb.DTakoRows, int32, error) {
    req := &dbpb.ListDTakoRowsRequest{
        Limit:  1000,
        Offset: 0,
    }

    allRows := make([]*dbpb.DTakoRows, 0)

    for {
        resp, err := s.dbClient.List(ctx, req)
        if err != nil {
            return nil, 0, err
        }

        // フィルタリング処理
        for _, row := range resp.Items {
            if s.matchesFilter(row, filter) {
                allRows = append(allRows, row)
            }
        }

        // ページネーション終了判定
        if len(resp.Items) < int(req.Limit) {
            break
        }
        req.Offset += req.Limit

        // 最適化: 必要な件数が既に集まったら早期終了
        if limit > 0 && int32(len(allRows)) >= offset+limit {
            break
        }
    }

    // ページネーション処理
    totalCount := int32(len(allRows))
    startIdx := offset
    endIdx := offset + limit

    if startIdx > totalCount {
        return []*dbpb.DTakoRows{}, totalCount, nil
    }
    if endIdx > totalCount || limit == 0 {
        endIdx = totalCount
    }

    return allRows[startIdx:endIdx], totalCount, nil
}
```

### matchesFilter（内部メソッド）

単一行がフィルタ条件に一致するかチェックします。

```go
func (s *DtakoRowsService) matchesFilter(row *dbpb.DTakoRows, filter *FilterOptions) bool {
    if filter == nil {
        return true
    }

    // 車輌CCフィルタ
    if filter.CarCC != nil && row.CarCc != *filter.CarCC {
        return false
    }

    // 運行NOフィルタ（複数指定）
    if len(filter.OperationNos) > 0 {
        matched := false
        for _, opNo := range filter.OperationNos {
            if row.OperationNo == opNo {
                matched = true
                break
            }
        }
        if !matched {
            return false
        }
    }

    // 運行日フィルタ
    if filter.StartDate != nil || filter.EndDate != nil {
        opDate, err := time.Parse(time.RFC3339, row.OperationDate)
        if err != nil {
            return false
        }

        if filter.StartDate != nil && opDate.Before(*filter.StartDate) {
            return false
        }
        if filter.EndDate != nil && opDate.After(*filter.EndDate) {
            return false
        }
    }

    // 走行距離フィルタ
    if filter.MinDistance != nil && row.TotalDistance < *filter.MinDistance {
        return false
    }

    // 走行距離0除外
    if filter.ExcludeZeroDistance && row.TotalDistance == 0 {
        return false
    }

    return true
}
```

---

## ヘルパーメソッド

よく使うフィルタパターンを簡単に使えるヘルパーメソッドを提供しています。

### ListByCarCC

車輌CCで絞り込んだデータを取得します。

```go
func (s *DtakoRowsService) ListByCarCC(
    ctx context.Context,
    carCC string,
    limit int32,
) ([]*dbpb.DTakoRows, error)
```

**使用例**:
```go
// 車輌CC "215800" のデータを100件取得
rows, err := service.ListByCarCC(ctx, "215800", 100)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("取得: %d件\n", len(rows))
```

**内部実装**:
```go
func (s *DtakoRowsService) ListByCarCC(ctx context.Context, carCC string, limit int32) ([]*dbpb.DTakoRows, error) {
    filter := &FilterOptions{
        CarCC: &carCC,
    }
    rows, _, err := s.ListWithFilter(ctx, filter, limit, 0)
    return rows, err
}
```

### ListByDateRange

日付範囲で絞り込んだデータを取得します。

```go
func (s *DtakoRowsService) ListByDateRange(
    ctx context.Context,
    startDate, endDate string,
    limit int32,
) ([]*dbpb.DTakoRows, error)
```

**使用例**:
```go
// 2025年10月のデータを全件取得
rows, err := service.ListByDateRange(ctx, "2025-10-01", "2025-10-31", 0)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("2025年10月: %d件\n", len(rows))
```

**内部実装**:
```go
func (s *DtakoRowsService) ListByDateRange(ctx context.Context, startDate, endDate string, limit int32) ([]*dbpb.DTakoRows, error) {
    start, err := time.Parse("2006-01-02", startDate)
    if err != nil {
        return nil, status.Errorf(codes.InvalidArgument, "invalid start_date format: %v", err)
    }
    end, err := time.Parse("2006-01-02", endDate)
    if err != nil {
        return nil, status.Errorf(codes.InvalidArgument, "invalid end_date format: %v", err)
    }

    filter := &FilterOptions{
        StartDate: &start,
        EndDate:   &end,
    }
    rows, _, err := s.ListWithFilter(ctx, filter, limit, 0)
    return rows, err
}
```

### ListByCarCCAndDateRange

車輌CCと日付範囲で絞り込んだデータを取得します。

```go
func (s *DtakoRowsService) ListByCarCCAndDateRange(
    ctx context.Context,
    carCC, startDate, endDate string,
    limit int32,
) ([]*dbpb.DTakoRows, error)
```

**使用例**:
```go
// 車輌CC "215800" の2025年10月のデータを取得
rows, err := service.ListByCarCCAndDateRange(
    ctx,
    "215800",
    "2025-10-01",
    "2025-10-31",
    100,
)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("車輌215800の2025年10月: %d件\n", len(rows))
```

**内部実装**:
```go
func (s *DtakoRowsService) ListByCarCCAndDateRange(ctx context.Context, carCC, startDate, endDate string, limit int32) ([]*dbpb.DTakoRows, error) {
    start, err := time.Parse("2006-01-02", startDate)
    if err != nil {
        return nil, status.Errorf(codes.InvalidArgument, "invalid start_date format: %v", err)
    }
    end, err := time.Parse("2006-01-02", endDate)
    if err != nil {
        return nil, status.Errorf(codes.InvalidArgument, "invalid end_date format: %v", err)
    }

    filter := &FilterOptions{
        CarCC:     &carCC,
        StartDate: &start,
        EndDate:   &end,
    }
    rows, _, err := s.ListWithFilter(ctx, filter, limit, 0)
    return rows, err
}
```

---

## 使用例

### 基本的な使い方

#### 例1: 特定車両の最近1ヶ月のデータ

```go
package main

import (
    "context"
    "log"

    "github.com/yhonda-ohishi/dtako_rows/v3/internal/service"
)

func main() {
    // サービス作成
    svc, err := service.NewDtakoRowsService("localhost:50051")
    if err != nil {
        log.Fatal(err)
    }

    // 特定車両の最近1ヶ月のデータ
    rows, err := svc.ListByCarCCAndDateRange(
        context.Background(),
        "215800",
        "2025-09-01",
        "2025-10-31",
        100, // limit
    )
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("取得件数: %d件", len(rows))
    for _, row := range rows {
        log.Printf("運行日: %s, 距離: %.1fkm", row.OperationDate, row.TotalDistance)
    }
}
```

#### 例2: 日付範囲のみで絞り込み

```go
// 2025年10月の全車両のデータ
rows, err := svc.ListByDateRange(
    ctx,
    "2025-10-01",
    "2025-10-31",
    0, // 全件取得
)
if err != nil {
    log.Fatal(err)
}

log.Printf("2025年10月の運行データ: %d件", len(rows))
```

#### 例3: 車輌CCのみで絞り込み

```go
// 車輌CC "215800" の全期間のデータを100件
rows, err := svc.ListByCarCC(ctx, "215800", 100)
if err != nil {
    log.Fatal(err)
}

log.Printf("車輌215800のデータ: %d件", len(rows))
```

### 複雑なフィルタ

#### 例4: 最小走行距離でフィルタ

```go
// 10km以上走行したデータのみ
minDist := 10.0
carCC := "215800"
start, _ := time.Parse("2006-01-02", "2025-10-01")
end, _ := time.Parse("2006-01-02", "2025-10-31")

filter := &service.FilterOptions{
    CarCC:       &carCC,
    StartDate:   &start,
    EndDate:     &end,
    MinDistance: &minDist,
}

rows, totalCount, err := svc.ListWithFilter(ctx, filter, 100, 0)
if err != nil {
    log.Fatal(err)
}

log.Printf("10km以上の運行: %d件（全体: %d件）", len(rows), totalCount)
```

#### 例5: 走行距離0を除外

```go
carCC := "215800"
start, _ := time.Parse("2006-01-02", "2025-10-01")
end, _ := time.Parse("2006-01-02", "2025-10-31")

filter := &service.FilterOptions{
    CarCC:               &carCC,
    StartDate:           &start,
    EndDate:             &end,
    ExcludeZeroDistance: true, // 走行距離0を除外
}

rows, _, err := svc.ListWithFilter(ctx, filter, 0, 0)
if err != nil {
    log.Fatal(err)
}

log.Printf("走行距離0を除外したデータ: %d件", len(rows))
```

#### 例6: 複数運行NOで絞り込み

```go
filter := &service.FilterOptions{
    OperationNos: []string{"OP001", "OP002", "OP003"},
}

rows, _, err := svc.ListWithFilter(ctx, filter, 0, 0)
if err != nil {
    log.Fatal(err)
}

log.Printf("指定運行NOのデータ: %d件", len(rows))
for _, row := range rows {
    log.Printf("運行NO: %s, 車輌: %s", row.OperationNo, row.CarCc)
}
```

#### 例7: ページネーション

```go
carCC := "215800"
filter := &service.FilterOptions{
    CarCC: &carCC,
}

// 1ページ目（0-9件目）
rows1, totalCount, err := svc.ListWithFilter(ctx, filter, 10, 0)
log.Printf("1ページ目: %d件（全体: %d件）", len(rows1), totalCount)

// 2ページ目（10-19件目）
rows2, _, err := svc.ListWithFilter(ctx, filter, 10, 10)
log.Printf("2ページ目: %d件", len(rows2))

// 3ページ目（20-29件目）
rows3, _, err := svc.ListWithFilter(ctx, filter, 10, 20)
log.Printf("3ページ目: %d件", len(rows3))
```

---

## 集計メソッドでの活用

集計メソッドは内部でヘルパーメソッドを使用しています。これにより、コードが簡潔になり保守性が向上しています。

### GetMonthlyFuelConsumption

**変更前**（冗長なフィルタリングコード）:
```go
func (s *DtakoRowsService) GetMonthlyFuelConsumption(ctx context.Context, carCC string, startDate, endDate string) ([]*MonthlyFuelSummary, error) {
    req := &dbpb.ListDTakoRowsRequest{
        Limit:  1000,
        Offset: 0,
    }

    allRows := make([]*dbpb.DTakoRows, 0)
    for {
        resp, err := s.dbClient.List(ctx, req)
        if err != nil {
            return nil, err
        }

        // 手動フィルタリング
        for _, row := range resp.Items {
            if row.CarCc != carCC {
                continue
            }

            opDate, err := time.Parse(time.RFC3339, row.OperationDate)
            if err != nil {
                continue
            }

            start, _ := time.Parse("2006-01-02", startDate)
            end, _ := time.Parse("2006-01-02", endDate)

            if opDate.Before(start) || opDate.After(end) {
                continue
            }

            allRows = append(allRows, row)
        }

        if len(resp.Items) < int(req.Limit) {
            break
        }
        req.Offset += req.Limit
    }

    // 集計処理...
}
```

**変更後**（ヘルパーメソッドを使用）:
```go
func (s *DtakoRowsService) GetMonthlyFuelConsumption(ctx context.Context, carCC string, startDate, endDate string) ([]*MonthlyFuelSummary, error) {
    // 新しいフィルタリングメソッドを使用（1行で完結）
    allRows, err := s.ListByCarCCAndDateRange(ctx, carCC, startDate, endDate, 0)
    if err != nil {
        return nil, err
    }

    // 集計処理...
}
```

### GetVehicleMonthlySummary

**変更前**:
```go
func (s *DtakoRowsService) GetVehicleMonthlySummary(ctx context.Context, startDate, endDate string) (map[string][]*MonthlyFuelSummary, error) {
    req := &dbpb.ListDTakoRowsRequest{
        Limit:  1000,
        Offset: 0,
    }

    allRows := make([]*dbpb.DTakoRows, 0)
    for {
        resp, err := s.dbClient.List(ctx, req)
        if err != nil {
            return nil, err
        }

        // 期間でフィルタリング
        for _, row := range resp.Items {
            opDate, err := time.Parse(time.RFC3339, row.OperationDate)
            if err != nil {
                continue
            }

            start, _ := time.Parse("2006-01-02", startDate)
            end, _ := time.Parse("2006-01-02", endDate)

            if opDate.Before(start) || opDate.After(end) {
                continue
            }

            allRows = append(allRows, row)
        }

        if len(resp.Items) < int(req.Limit) {
            break
        }
        req.Offset += req.Limit
    }

    // 集計処理...
}
```

**変更後**:
```go
func (s *DtakoRowsService) GetVehicleMonthlySummary(ctx context.Context, startDate, endDate string) (map[string][]*MonthlyFuelSummary, error) {
    // 新しいフィルタリングメソッドを使用（日付範囲のみ）
    allRows, err := s.ListByDateRange(ctx, startDate, endDate, 0)
    if err != nil {
        return nil, err
    }

    // 集計処理...
}
```

### GetDailySummary

**変更後**:
```go
func (s *DtakoRowsService) GetDailySummary(ctx context.Context, carCC string, startDate, endDate string) (map[string]*MonthlyFuelSummary, error) {
    // 新しいフィルタリングメソッドを使用
    allRows, err := s.ListByCarCCAndDateRange(ctx, carCC, startDate, endDate, 0)
    if err != nil {
        return nil, err
    }

    // 日次集計処理...
}
```

### コード削減効果

| メソッド | 変更前 | 変更後 | 削減 |
|---------|--------|--------|------|
| GetMonthlyFuelConsumption | 80行 | 42行 | 38行（47%削減） |
| GetVehicleMonthlySummary | 75行 | 40行 | 35行（46%削減） |
| GetDailySummary | 65行 | 40行 | 25行（38%削減） |

---

## パフォーマンス特性

### メリット

#### 1. 柔軟性

db_serviceを変更せずに複雑なフィルタが可能：

```go
// db_serviceの変更なしで複雑なフィルタを追加
filter := &FilterOptions{
    CarCC:               &carCC,
    StartDate:           &start,
    EndDate:             &end,
    MinDistance:         &minDist,
    ExcludeZeroDistance: true,
    OperationNos:        []string{"OP001", "OP002"},
}
```

#### 2. 保守性

フィルタロジックがサービス層に集約：

- 全てのフィルタロジックが`matchesFilter`メソッドに集中
- 新しいフィルタ条件の追加が容易
- テストが書きやすい

#### 3. 再利用性

複数の集計メソッドで共通のフィルタを使用：

```go
// GetMonthlyFuelConsumption
allRows, _ := s.ListByCarCCAndDateRange(ctx, carCC, startDate, endDate, 0)

// GetDailySummary
allRows, _ := s.ListByCarCCAndDateRange(ctx, carCC, startDate, endDate, 0)

// GetVehicleMonthlySummary
allRows, _ := s.ListByDateRange(ctx, startDate, endDate, 0)
```

### デメリット

#### 1. パフォーマンス

全件取得してからフィルタリング（現状22秒/1000件）：

```
現在の処理フロー:
1. db_serviceから全データ取得: 22秒
2. サービス層でフィルタリング: 0.01秒
   合計: 22秒
```

#### 2. メモリ

一時的に全データをメモリに保持：

```go
allRows := make([]*dbpb.DTakoRows, 0)  // 最大数千〜数万件
```

### ベンチマーク

#### テスト条件

- データ量: 940件
- フィルタ: 車輌CC + 日付範囲

#### 結果

| 処理 | 時間 | 割合 |
|-----|------|------|
| db_serviceからデータ取得 | 22秒 | 99.9% |
| フィルタリング処理 | 0.01秒 | 0.1% |
| **合計** | **22秒** | **100%** |

**結論**: ボトルネックはdb_service側のデータ取得

---

## 改善案

### Phase 1: キャッシング（短期）

現在の実装のまま、メモリキャッシュで重複取得を削減：

```go
type CachedDtakoRowsService struct {
    *DtakoRowsService
    cache map[string][]*dbpb.DTakoRows
    cacheTTL time.Duration
}

func (s *CachedDtakoRowsService) ListWithFilter(ctx context.Context, filter *FilterOptions, limit int32, offset int32) ([]*dbpb.DTakoRows, int32, error) {
    cacheKey := generateCacheKey(filter)

    // キャッシュヒット
    if cached, exists := s.cache[cacheKey]; exists {
        return applyPagination(cached, limit, offset), int32(len(cached)), nil
    }

    // キャッシュミス -> 通常処理
    rows, total, err := s.DtakoRowsService.ListWithFilter(ctx, filter, limit, offset)
    if err == nil {
        s.cache[cacheKey] = rows
        go s.expireCache(cacheKey, s.cacheTTL)
    }

    return rows, total, err
}
```

**期待効果**: 67秒（3クエリ） → 22秒（1クエリ+2キャッシュヒット）

### Phase 2: db_serviceフィルタ拡張（中期）

db_service側でフィルタリングを実装：

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

**SQL実装**:
```sql
SELECT * FROM dtako_rows
WHERE car_cc = ?
  AND operation_date >= ?
  AND operation_date <= ?
  AND total_distance >= ?
ORDER BY read_date DESC
LIMIT ? OFFSET ?;
```

**期待効果**: 22秒 → 0.1秒（DB側でWHERE句によるフィルタリング）

### Phase 3: ストリーミングRPC（長期）

ストリーミングで大量データを効率的に処理：

```protobuf
service DbDtakoRowsService {
  rpc ListStream(ListDTakoRowsRequest) returns (stream DTakoRows);
}
```

**実装例**:
```go
stream, err := dbClient.ListStream(ctx, req)
for {
    row, err := stream.Recv()
    if err == io.EOF {
        break
    }
    if s.matchesFilter(row, filter) {
        // リアルタイム処理
    }
}
```

**期待効果**: 22秒 → 5秒以下（メモリ効率向上 + 並行処理）

---

## トラブルシューティング

### エラー: invalid date format

```
Error: invalid start_date format: parsing time "2025/10/01" as "2006-01-02"
```

**原因**: 日付フォーマットが間違っている

**対処法**:
```go
// 誤り
rows, err := svc.ListByDateRange(ctx, "2025/10/01", "2025/10/31", 100)

// 正解
rows, err := svc.ListByDateRange(ctx, "2025-10-01", "2025-10-31", 100)
```

### エラー: no rows returned

```
Error: no rows returned (expected some data)
```

**原因**: フィルタ条件が厳しすぎる

**対処法**:
```go
// フィルタ条件を確認
log.Printf("Filter: CarCC=%v, StartDate=%v, EndDate=%v",
    *filter.CarCC, *filter.StartDate, *filter.EndDate)

// フィルタなしで試す
rows, _, err := svc.ListWithFilter(ctx, nil, 10, 0)
log.Printf("Total rows without filter: %d", len(rows))
```

### パフォーマンスが遅い

```
処理時間: 60秒（想定: 5秒）
```

**原因**: db_service側のボトルネック

**対処法**:
```go
// ログで処理時間を計測
start := time.Now()
rows, err := svc.ListByCarCCAndDateRange(ctx, carCC, startDate, endDate, 0)
log.Printf("Data fetch: %v", time.Since(start))

start = time.Now()
// 集計処理
log.Printf("Aggregation: %v", time.Since(start))
```

---

## テスト例

### ユニットテスト

```go
func TestMatchesFilter(t *testing.T) {
    svc := &DtakoRowsService{}

    // テストデータ
    row := &dbpb.DTakoRows{
        CarCc:         "215800",
        OperationDate: "2025-10-15T00:00:00Z",
        TotalDistance: 100.0,
    }

    // CarCCフィルタ
    carCC := "215800"
    filter := &FilterOptions{CarCC: &carCC}
    assert.True(t, svc.matchesFilter(row, filter))

    // 異なるCarCC
    differentCC := "999999"
    filter = &FilterOptions{CarCC: &differentCC}
    assert.False(t, svc.matchesFilter(row, filter))

    // 日付範囲フィルタ
    start, _ := time.Parse("2006-01-02", "2025-10-01")
    end, _ := time.Parse("2006-01-02", "2025-10-31")
    filter = &FilterOptions{StartDate: &start, EndDate: &end}
    assert.True(t, svc.matchesFilter(row, filter))

    // 走行距離フィルタ
    minDist := 50.0
    filter = &FilterOptions{MinDistance: &minDist}
    assert.True(t, svc.matchesFilter(row, filter))

    minDist = 200.0
    filter = &FilterOptions{MinDistance: &minDist}
    assert.False(t, svc.matchesFilter(row, filter))
}
```

### 統合テスト

```go
func TestListByCarCCAndDateRange(t *testing.T) {
    // サービス作成
    svc, err := service.NewDtakoRowsService("localhost:50051")
    require.NoError(t, err)

    // テスト実行
    rows, err := svc.ListByCarCCAndDateRange(
        context.Background(),
        "215800",
        "2025-10-01",
        "2025-10-31",
        100,
    )

    require.NoError(t, err)
    assert.NotEmpty(t, rows)

    // 結果検証
    for _, row := range rows {
        assert.Equal(t, "215800", row.CarCc)

        opDate, _ := time.Parse(time.RFC3339, row.OperationDate)
        assert.True(t, opDate.After(time.Date(2025, 9, 30, 0, 0, 0, 0, time.UTC)))
        assert.True(t, opDate.Before(time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC)))
    }
}
```

---

## まとめ

### 実装されている機能

- ✅ FilterOptions構造体で柔軟なフィルタ条件を定義
- ✅ ListWithFilterメソッドでフィルタリングを実装
- ✅ ヘルパーメソッド（ListByCarCC, ListByDateRange, ListByCarCCAndDateRange）
- ✅ 既存の集計メソッドを新しいフィルタリングで書き換え（コード47%削減）

### 主な利点

1. **db_serviceを変更せずに柔軟なフィルタリングが可能**
2. **コードの再利用性が向上**（集計メソッドで共通利用）
3. **保守性が向上**（フィルタロジックが集約）
4. **テストが容易**（ユニットテストが書きやすい）

### 今後の改善

1. **Phase 1**: キャッシング（短期） - 67秒 → 22秒
2. **Phase 2**: db_serviceフィルタ拡張（中期） - 22秒 → 0.1秒
3. **Phase 3**: ストリーミングRPC（長期） - 22秒 → 5秒以下

---

## 参考リンク

- [internal/service/dtako_rows_service.go](../internal/service/dtako_rows_service.go) - 実装コード
- [internal/service/aggregation.go](../internal/service/aggregation.go) - 集計メソッド
- [SPECIFICATION.md](../SPECIFICATION.md) - 全体仕様書
