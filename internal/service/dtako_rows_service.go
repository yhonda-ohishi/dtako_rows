package service

import (
	"context"

	pb "github.com/yhonda-ohishi/dtako_rows/v3/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// DtakoRowsService gRPCサービス実装（読み取り専用）
// データアクセスはdb_service経由で行う
type DtakoRowsService struct {
	pb.UnimplementedDtakoRowsServiceServer
	// TODO: db_serviceクライアントを追加
}

// NewDtakoRowsService サービスの作成
func NewDtakoRowsService() *DtakoRowsService {
	return &DtakoRowsService{}
}

// GetRow 運行データ取得
func (s *DtakoRowsService) GetRow(ctx context.Context, req *pb.GetRowRequest) (*pb.GetRowResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	// TODO: db_service経由でデータ取得
	return nil, status.Error(codes.Unimplemented, "not implemented yet - requires db_service integration")
}

// ListRows 運行データ一覧取得
func (s *DtakoRowsService) ListRows(ctx context.Context, req *pb.ListRowsRequest) (*pb.ListRowsResponse, error) {
	// TODO: db_service経由でデータ取得
	return nil, status.Error(codes.Unimplemented, "not implemented yet - requires db_service integration")
}

// SearchRows 運行データ検索
func (s *DtakoRowsService) SearchRows(ctx context.Context, req *pb.SearchRowsRequest) (*pb.ListRowsResponse, error) {
	// TODO: db_service経由でデータ検索
	return nil, status.Error(codes.Unimplemented, "not implemented yet - requires db_service integration")
}
