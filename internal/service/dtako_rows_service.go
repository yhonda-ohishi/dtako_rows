package service

import (
	"context"
	"time"

	"github.com/yhonda-ohishi/dtako_rows/internal/models"
	"github.com/yhonda-ohishi/dtako_rows/internal/repository"
	pb "github.com/yhonda-ohishi/dtako_rows/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// DtakoRowsService gRPCサービス実装
type DtakoRowsService struct {
	pb.UnimplementedDtakoRowsServiceServer
	repo repository.DtakoRowRepository
}

// NewDtakoRowsService サービスの作成
func NewDtakoRowsService(repo repository.DtakoRowRepository) *DtakoRowsService {
	return &DtakoRowsService{
		repo: repo,
	}
}

// GetRow 運行データ取得
func (s *DtakoRowsService) GetRow(ctx context.Context, req *pb.GetRowRequest) (*pb.GetRowResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	row, err := s.repo.GetByID(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "row not found: %v", err)
	}

	return &pb.GetRowResponse{
		Row: modelToProto(row),
	}, nil
}

// ListRows 運行データ一覧取得
func (s *DtakoRowsService) ListRows(ctx context.Context, req *pb.ListRowsRequest) (*pb.ListRowsResponse, error) {
	page := req.Page
	if page <= 0 {
		page = 1
	}

	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 50
	}

	rows, total, err := s.repo.List(ctx, int(page), int(pageSize), req.OrderBy)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list rows: %v", err)
	}

	protoRows := make([]*pb.DtakoRow, len(rows))
	for i, row := range rows {
		protoRows[i] = modelToProto(row)
	}

	return &pb.ListRowsResponse{
		Rows:     protoRows,
		Total:    int32(total),
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// CreateRow 運行データ作成
func (s *DtakoRowsService) CreateRow(ctx context.Context, req *pb.CreateRowRequest) (*pb.CreateRowResponse, error) {
	if req.Row == nil {
		return nil, status.Error(codes.InvalidArgument, "row is required")
	}

	row := protoToModel(req.Row)
	if err := s.repo.Create(ctx, row); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create row: %v", err)
	}

	return &pb.CreateRowResponse{
		Row: modelToProto(row),
	}, nil
}

// UpdateRow 運行データ更新
func (s *DtakoRowsService) UpdateRow(ctx context.Context, req *pb.UpdateRowRequest) (*pb.UpdateRowResponse, error) {
	if req.Row == nil {
		return nil, status.Error(codes.InvalidArgument, "row is required")
	}

	if req.Row.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	row := protoToModel(req.Row)
	if err := s.repo.Update(ctx, row); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update row: %v", err)
	}

	return &pb.UpdateRowResponse{
		Row: modelToProto(row),
	}, nil
}

// DeleteRow 運行データ削除
func (s *DtakoRowsService) DeleteRow(ctx context.Context, req *pb.DeleteRowRequest) (*pb.DeleteRowResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	if err := s.repo.Delete(ctx, req.Id); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete row: %v", err)
	}

	return &pb.DeleteRowResponse{
		Success: true,
	}, nil
}

// SearchRows 運行データ検索
func (s *DtakoRowsService) SearchRows(ctx context.Context, req *pb.SearchRowsRequest) (*pb.ListRowsResponse, error) {
	var dateFrom, dateTo *timestamppb.Timestamp
	if req.DateFrom != nil {
		dateFrom = req.DateFrom
	}
	if req.DateTo != nil {
		dateTo = req.DateTo
	}

	var dateFromTime, dateToTime *time.Time
	if dateFrom != nil {
		t := dateFrom.AsTime()
		dateFromTime = &t
	}
	if dateTo != nil {
		t := dateTo.AsTime()
		dateToTime = &t
	}

	rows, err := s.repo.Search(ctx, dateFromTime, dateToTime, req.SharyouCc, req.JomuinCd1)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to search rows: %v", err)
	}

	protoRows := make([]*pb.DtakoRow, len(rows))
	for i, row := range rows {
		protoRows[i] = modelToProto(row)
	}

	return &pb.ListRowsResponse{
		Rows:     protoRows,
		Total:    int32(len(rows)),
		Page:     1,
		PageSize: int32(len(rows)),
	}, nil
}

// modelToProto モデルをProtobufに変換
func modelToProto(row *models.DtakoRow) *pb.DtakoRow {
	protoRow := &pb.DtakoRow{
		Id:                  row.ID,
		UnkoNo:              row.UnkoNo,
		SharyouCc:           row.SharyouCC,
		JomuinCd1:           row.JomuinCD1,
		ShukkoDatetime:      timestamppb.New(row.ShukkoDateTime),
		UnkoDate:            timestamppb.New(row.UnkoDate),
		TaishouJomuinKubun:  int32(row.TaishouJomuinKubun),
		SoukouKyori:         row.SoukouKyori,
		NenryouShiyou:       row.NenryouShiyou,
		CreatedAt:           timestamppb.New(row.Created),
		UpdatedAt:           timestamppb.New(row.Modified),
	}

	if row.KikoDateTime != nil {
		protoRow.KikoDatetime = timestamppb.New(*row.KikoDateTime)
	}

	return protoRow
}

// protoToModel Protobufをモデルに変換
func protoToModel(row *pb.DtakoRow) *models.DtakoRow {
	model := &models.DtakoRow{
		ID:                   row.Id,
		UnkoNo:               row.UnkoNo,
		SharyouCC:            row.SharyouCc,
		JomuinCD1:            row.JomuinCd1,
		ShukkoDateTime:       row.ShukkoDatetime.AsTime(),
		UnkoDate:             row.UnkoDate.AsTime(),
		TaishouJomuinKubun:   int(row.TaishouJomuinKubun),
		SoukouKyori:          row.SoukouKyori,
		NenryouShiyou:        row.NenryouShiyou,
	}

	if row.KikoDatetime != nil {
		t := row.KikoDatetime.AsTime()
		model.KikoDateTime = &t
	}

	if row.CreatedAt != nil {
		model.Created = row.CreatedAt.AsTime()
	}

	if row.UpdatedAt != nil {
		model.Modified = row.UpdatedAt.AsTime()
	}

	return model
}
