package service

import (
	"context"
	"fmt"
	"log"

	dbgrpc "buf.build/gen/go/yhonda-ohishi/db-service/grpc/go/_gogrpc"
	pb "github.com/yhonda-ohishi/dtako_rows/v3/proto"
)

// DtakoRowsAggregationService 集計サービス実装
type DtakoRowsAggregationService struct {
	pb.UnimplementedDtakoRowsAggregationServiceServer
	dbClient dbgrpc.DTakoRowsServiceClient
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
func NewDtakoRowsAggregationServiceWithClient(client dbgrpc.DTakoRowsServiceClient) *DtakoRowsAggregationService {
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
