package service

import (
	"context"
	"log"
	"time"

	dbpb "buf.build/gen/go/yhonda-ohishi/db-service/protocolbuffers/go"
	dbgrpc "buf.build/gen/go/yhonda-ohishi/db-service/grpc/go/_gogrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// FilterOptions サービス層でのフィルタリングオプション
type FilterOptions struct {
	CarCC              *string    // 車輌CC（完全一致）
	StartDate          *time.Time // 運行開始日（以降）
	EndDate            *time.Time // 運行終了日（以前）
	MinDistance        *float64   // 最小走行距離
	OperationNos       []string   // 運行NO（複数指定可）
	ExcludeZeroDistance bool      // 走行距離0のデータを除外
}

// DtakoRowsService gRPCサービス実装（読み取り専用）
// データアクセスはdb_service経由で行う
type DtakoRowsService struct {
	dbgrpc.UnimplementedDb_DTakoRowsServiceServer
	dbClient dbgrpc.Db_DTakoRowsServiceClient
}

// NewDtakoRowsService サービスの作成（スタンドアロン用）
// 外部のdb_serviceにgRPC接続する
func NewDtakoRowsService(dbServiceAddr string) (*DtakoRowsService, error) {
	// db_serviceへの接続
	conn, err := grpc.NewClient(dbServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	client := dbgrpc.NewDb_DTakoRowsServiceClient(conn)
	log.Printf("Connected to db_service at %s", dbServiceAddr)

	return &DtakoRowsService{
		dbClient: client,
	}, nil
}

// NewDtakoRowsServiceWithClient サービスの作成（desktop-server統合用）
// 既存のdb_serviceクライアントを受け取る
func NewDtakoRowsServiceWithClient(client dbgrpc.Db_DTakoRowsServiceClient) *DtakoRowsService {
	log.Println("Creating dtako_rows service with existing db_service client")
	return &DtakoRowsService{
		dbClient: client,
	}
}

// Get 運行データ取得
func (s *DtakoRowsService) Get(ctx context.Context, req *dbpb.Db_GetDTakoRowsRequest) (*dbpb.Db_DTakoRowsResponse, error) {
	// ビジネスロジック: バリデーション
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	log.Printf("Get request: id=%s", req.Id)

	// db_service経由でデータ取得
	resp, err := s.dbClient.Get(ctx, req)
	if err != nil {
		log.Printf("Failed to get row: %v", err)
		return nil, err
	}

	// ビジネスロジック: データ検証
	if resp.DtakoRows == nil {
		return nil, status.Error(codes.NotFound, "row not found")
	}

	log.Printf("Retrieved row: id=%s, operation_no=%s", resp.DtakoRows.Id, resp.DtakoRows.OperationNo)
	return resp, nil
}

// List 運行データ一覧取得
func (s *DtakoRowsService) List(ctx context.Context, req *dbpb.Db_ListDTakoRowsRequest) (*dbpb.Db_ListDTakoRowsResponse, error) {
	// ビジネスロジック: デフォルトのlimitとorder_byを設定
	if req.Limit == 0 {
		req.Limit = 100 // デフォルト100件
	}
	if req.Limit > 1000 {
		req.Limit = 1000 // 最大1000件に制限
	}
	if req.OrderBy == nil || *req.OrderBy == "" {
		defaultOrderBy := "read_date DESC" // デフォルトソート：読取日降順
		req.OrderBy = &defaultOrderBy
	}

	log.Printf("List request: limit=%d, offset=%d, order_by=%s", req.Limit, req.Offset, *req.OrderBy)

	// db_service経由でデータ取得
	resp, err := s.dbClient.List(ctx, req)
	if err != nil {
		log.Printf("Failed to list rows: %v", err)
		return nil, err
	}

	// ビジネスロジック: データの後処理
	// 1. 走行距離でフィルタリング
	if req.Limit == 1 && req.Offset == 0 && req.OrderBy != nil && *req.OrderBy == "" {
		// 特殊なリクエスト形式で集計モードを判定（将来的にはproto拡張が必要）
		filteredItems := make([]*dbpb.Db_DTakoRows, 0)
		for _, item := range resp.Items {
			if item.TotalDistance > 0 {
				filteredItems = append(filteredItems, item)
			}
		}
		resp.Items = filteredItems
		resp.TotalCount = int32(len(filteredItems))
	}

	log.Printf("Retrieved %d rows (total: %d)", len(resp.Items), resp.TotalCount)
	return resp, nil
}

// GetByOperationNo 運行NOで運行データ取得
func (s *DtakoRowsService) GetByOperationNo(ctx context.Context, req *dbpb.Db_GetDTakoRowsByOperationNoRequest) (*dbpb.Db_ListDTakoRowsResponse, error) {
	// ビジネスロジック: バリデーション
	if req.OperationNo == "" {
		return nil, status.Error(codes.InvalidArgument, "operation_no is required")
	}

	log.Printf("GetByOperationNo request: operation_no=%s", req.OperationNo)

	// db_service経由でデータ取得
	resp, err := s.dbClient.GetByOperationNo(ctx, req)
	if err != nil {
		log.Printf("Failed to get rows by operation_no: %v", err)
		return nil, err
	}

	log.Printf("Retrieved %d rows for operation_no=%s", len(resp.Items), req.OperationNo)
	return resp, nil
}

// ListWithFilter フィルタリングオプション付きデータ取得
//
// サービス層でフィルタリングを行います。
// db_serviceにフィルタ機能がない場合でも、このメソッドで柔軟なフィルタリングが可能です。
func (s *DtakoRowsService) ListWithFilter(ctx context.Context, filter *FilterOptions, limit int32, offset int32) ([]*dbpb.Db_DTakoRows, int32, error) {
	log.Printf("ListWithFilter: limit=%d, offset=%d", limit, offset)

	// ページネーションで全データを取得
	req := &dbpb.Db_ListDTakoRowsRequest{
		Limit:  1000, // 大きめのバッチサイズ
		Offset: 0,
	}

	allRows := make([]*dbpb.Db_DTakoRows, 0)
	totalFetched := int32(0)

	for {
		resp, err := s.dbClient.List(ctx, req)
		if err != nil {
			log.Printf("Failed to list rows: %v", err)
			return nil, 0, err
		}

		// フィルタリング処理
		for _, row := range resp.Items {
			if s.matchesFilter(row, filter) {
				allRows = append(allRows, row)
			}
		}

		totalFetched += int32(len(resp.Items))

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

	log.Printf("Filtered %d rows from %d total rows", len(allRows), totalFetched)

	// ページネーション処理
	totalCount := int32(len(allRows))
	startIdx := offset
	endIdx := offset + limit

	if startIdx > totalCount {
		return []*dbpb.Db_DTakoRows{}, totalCount, nil
	}
	if endIdx > totalCount {
		endIdx = totalCount
	}
	if limit == 0 {
		endIdx = totalCount
	}

	return allRows[startIdx:endIdx], totalCount, nil
}

// matchesFilter 単一行がフィルタ条件に一致するかチェック
func (s *DtakoRowsService) matchesFilter(row *dbpb.Db_DTakoRows, filter *FilterOptions) bool {
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
			log.Printf("Failed to parse operation date: %s", row.OperationDate)
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

// ListByCarCC 車輌CCで絞り込んだデータ取得（ヘルパーメソッド）
func (s *DtakoRowsService) ListByCarCC(ctx context.Context, carCC string, limit int32) ([]*dbpb.Db_DTakoRows, error) {
	filter := &FilterOptions{
		CarCC: &carCC,
	}
	rows, _, err := s.ListWithFilter(ctx, filter, limit, 0)
	return rows, err
}

// ListByDateRange 日付範囲で絞り込んだデータ取得（ヘルパーメソッド）
func (s *DtakoRowsService) ListByDateRange(ctx context.Context, startDate, endDate string, limit int32) ([]*dbpb.Db_DTakoRows, error) {
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

// ListByCarCCAndDateRange 車輌CCと日付範囲で絞り込んだデータ取得（ヘルパーメソッド）
func (s *DtakoRowsService) ListByCarCCAndDateRange(ctx context.Context, carCC, startDate, endDate string, limit int32) ([]*dbpb.Db_DTakoRows, error) {
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
