package service

import (
	"context"
	"fmt"
	"log"

	dbpb "github.com/yhonda-ohishi/db_service/src/proto"
	pb "github.com/yhonda-ohishi/dtako_rows/v3/proto"
)

// DtakoRowsAggregationService 集計サービス実装
type DtakoRowsAggregationService struct {
	pb.UnimplementedDtakoRowsServiceServer
	dbClient dbpb.Db_DTakoRowsServiceClient
}

// NewDtakoRowsAggregationService 集計サービスの作成（スタンドアロン用）
func NewDtakoRowsAggregationService(dbServiceAddr string) (*DtakoRowsAggregationService, error) {
	// DtakoRowsServiceを作成してdb_clientを取得
	rowsService, err := NewDtakoRowsService(dbServiceAddr)
	if err != nil {
		return nil, err
	}

	return &DtakoRowsAggregationService{
		dbClient: rowsService.dbClient,
	}, nil
}

// NewDtakoRowsAggregationServiceWithClient 集計サービスの作成（desktop-server統合用）
func NewDtakoRowsAggregationServiceWithClient(client dbpb.Db_DTakoRowsServiceClient) *DtakoRowsAggregationService {
	log.Println("Creating dtako_rows aggregation service with existing db_service client")
	return &DtakoRowsAggregationService{
		dbClient: client,
	}
}

// GetMonthlyFuelConsumption 月次給油量集計
func (s *DtakoRowsAggregationService) GetMonthlyFuelConsumption(ctx context.Context, req *pb.GetMonthlyFuelConsumptionRequest) (*pb.MonthlyFuelConsumptionResponse, error) {
	log.Printf("GetMonthlyFuelConsumption: car_cc=%s, start=%s, end=%s", req.CarCc, req.StartDate, req.EndDate)

	// aggregation.goの関数を使って集計
	// 一時的にDtakoRowsServiceを作成
	rowsService := &DtakoRowsService{dbClient: s.dbClient}
	summaries, err := rowsService.GetMonthlyFuelConsumption(ctx, req.CarCc, req.StartDate, req.EndDate)
	if err != nil {
		return nil, err
	}

	// 内部型からproto型に変換
	pbSummaries := make([]*pb.MonthlyFuelSummary, len(summaries))
	for i, s := range summaries {
		avgFuelEfficiency := 0.0
		if s.TotalFuel > 0 {
			avgFuelEfficiency = s.TotalDistance / s.TotalFuel
		}

		pbSummaries[i] = &pb.MonthlyFuelSummary{
			CarCc:             s.CarCC,
			YearMonth:         s.YearMonth,
			TotalDistance:     s.TotalDistance,
			TotalFuel:         s.TotalFuel,
			TripCount:         s.TripCount,
			AvgFuelEfficiency: avgFuelEfficiency,
		}
	}

	return &pb.MonthlyFuelConsumptionResponse{
		Summaries: pbSummaries,
		CarCc:     req.CarCc,
		Period:    fmt.Sprintf("%s ~ %s", req.StartDate, req.EndDate),
	}, nil
}

// GetVehicleMonthlySummary 全車両月次サマリー
func (s *DtakoRowsAggregationService) GetVehicleMonthlySummary(ctx context.Context, req *pb.GetVehicleMonthlySummaryRequest) (*pb.VehicleMonthlySummaryResponse, error) {
	log.Printf("GetVehicleMonthlySummary: start=%s, end=%s", req.StartDate, req.EndDate)

	rowsService := &DtakoRowsService{dbClient: s.dbClient}
	summariesMap, err := rowsService.GetVehicleMonthlySummary(ctx, req.StartDate, req.EndDate)
	if err != nil {
		return nil, err
	}

	// 内部型からproto型に変換
	vehicleSummaries := make([]*pb.VehicleMonthlySummaries, 0, len(summariesMap))
	for carCC, summaries := range summariesMap {
		pbSummaries := make([]*pb.MonthlyFuelSummary, len(summaries))
		for i, s := range summaries {
			avgFuelEfficiency := 0.0
			if s.TotalFuel > 0 {
				avgFuelEfficiency = s.TotalDistance / s.TotalFuel
			}

			pbSummaries[i] = &pb.MonthlyFuelSummary{
				CarCc:             s.CarCC,
				YearMonth:         s.YearMonth,
				TotalDistance:     s.TotalDistance,
				TotalFuel:         s.TotalFuel,
				TripCount:         s.TripCount,
				AvgFuelEfficiency: avgFuelEfficiency,
			}
		}

		vehicleSummaries = append(vehicleSummaries, &pb.VehicleMonthlySummaries{
			CarCc:     carCC,
			Summaries: pbSummaries,
		})
	}

	return &pb.VehicleMonthlySummaryResponse{
		VehicleSummaries: vehicleSummaries,
		TotalVehicles:    int32(len(vehicleSummaries)),
		Period:           fmt.Sprintf("%s ~ %s", req.StartDate, req.EndDate),
	}, nil
}

// GetDailySummary 日次サマリー
func (s *DtakoRowsAggregationService) GetDailySummary(ctx context.Context, req *pb.GetDailySummaryRequest) (*pb.DailySummaryResponse, error) {
	log.Printf("GetDailySummary: car_cc=%s, start=%s, end=%s", req.CarCc, req.StartDate, req.EndDate)

	rowsService := &DtakoRowsService{dbClient: s.dbClient}
	dailyData, err := rowsService.GetDailySummary(ctx, req.CarCc, req.StartDate, req.EndDate)
	if err != nil {
		return nil, err
	}

	// 内部型からproto型に変換
	pbSummaries := make([]*pb.DailySummary, 0, len(dailyData))
	for date, s := range dailyData {
		pbSummaries = append(pbSummaries, &pb.DailySummary{
			CarCc:         s.CarCC,
			Date:          date,
			TotalDistance: s.TotalDistance,
			TotalFuel:     s.TotalFuel,
			TripCount:     s.TripCount,
		})
	}

	return &pb.DailySummaryResponse{
		Summaries: pbSummaries,
		CarCc:     req.CarCc,
		Period:    fmt.Sprintf("%s ~ %s", req.StartDate, req.EndDate),
	}, nil
}

// ExportMonthlyFuelCSV CSV形式でエクスポート
func (s *DtakoRowsAggregationService) ExportMonthlyFuelCSV(ctx context.Context, req *pb.GetMonthlyFuelConsumptionRequest) (*pb.ExportCSVResponse, error) {
	log.Printf("ExportMonthlyFuelCSV: car_cc=%s", req.CarCc)

	// 月次データを取得
	resp, err := s.GetMonthlyFuelConsumption(ctx, req)
	if err != nil {
		return nil, err
	}

	// CSV形式に変換
	csv := "年月,車両CC,走行距離(km),給油量(L),運行回数,平均燃費(km/L)\n"
	for _, s := range resp.Summaries {
		csv += fmt.Sprintf("%s,%s,%.1f,%.1f,%d,%.2f\n",
			s.YearMonth, s.CarCc, s.TotalDistance, s.TotalFuel, s.TripCount, s.AvgFuelEfficiency)
	}

	filename := fmt.Sprintf("monthly_fuel_%s_%s_%s.csv", req.CarCc, req.StartDate, req.EndDate)

	return &pb.ExportCSVResponse{
		CsvData:  csv,
		Filename: filename,
	}, nil
}

// GetRow 運行データ取得（db_serviceプロキシ）
func (s *DtakoRowsAggregationService) GetRow(ctx context.Context, req *pb.GetRowRequest) (*pb.RowResponse, error) {
	log.Printf("GetRow (proxy): id=%s", req.Id)

	// db_serviceから取得
	dbResp, err := s.dbClient.Get(ctx, &dbpb.Db_GetDTakoRowsRequest{
		Id: req.Id,
	})
	if err != nil {
		return nil, err
	}

	// db_serviceの型からdtako_rowsの型に変換
	row := convertDbRowToProto(dbResp.DtakoRows)

	return &pb.RowResponse{
		Row: row,
	}, nil
}

// ListRows 運行データ一覧取得（db_serviceプロキシ）
func (s *DtakoRowsAggregationService) ListRows(ctx context.Context, req *pb.ListRowsRequest) (*pb.ListRowsResponse, error) {
	log.Printf("ListRows (proxy): limit=%d, offset=%d", req.Limit, req.Offset)

	// db_serviceから取得
	dbResp, err := s.dbClient.List(ctx, &dbpb.Db_ListDTakoRowsRequest{
		Limit:   req.Limit,
		Offset:  req.Offset,
		OrderBy: req.OrderBy,
	})
	if err != nil {
		return nil, err
	}

	// db_serviceの型からdtako_rowsの型に変換
	rows := make([]*pb.Row, len(dbResp.Items))
	for i, dbRow := range dbResp.Items {
		rows[i] = convertDbRowToProto(dbRow)
	}

	return &pb.ListRowsResponse{
		Rows:       rows,
		TotalCount: dbResp.TotalCount,
	}, nil
}

// convertDbRowToProto db_serviceの運行データ型をdtako_rowsの型に変換
func convertDbRowToProto(dbRow *dbpb.Db_DTakoRows) *pb.Row {
	return &pb.Row{
		Id:                    dbRow.Id,
		OperationNo:           dbRow.OperationNo,
		ReadDate:              dbRow.ReadDate,
		OperationDate:         dbRow.OperationDate,
		CarCode:               dbRow.CarCode,
		CarCc:                 dbRow.CarCc,
		StartWorkDatetime:     dbRow.StartWorkDatetime,
		EndWorkDatetime:       dbRow.EndWorkDatetime,
		DepartureDatetime:     dbRow.DepartureDatetime,
		ReturnDatetime:        dbRow.ReturnDatetime,
		DepartureMeter:        dbRow.DepartureMeter,
		ReturnMeter:           dbRow.ReturnMeter,
		TotalDistance:         dbRow.TotalDistance,
		DriverCode1:           dbRow.DriverCode1,
		LoadedDistance:        dbRow.LoadedDistance,
		DestinationCityName:   dbRow.DestinationCityName,
		DestinationPlaceName:  dbRow.DestinationPlaceName,
	}
}
