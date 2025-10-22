package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/yhonda-ohishi/dtako_rows/v3/internal/service"
	pb "github.com/yhonda-ohishi/dtako_rows/v3/proto"
)

func main() {
	// 集計サービスを作成
	aggService, err := service.NewDtakoRowsAggregationService("localhost:50051")
	if err != nil {
		log.Fatalf("Failed to create service: %v", err)
	}

	// 最近1ヶ月の日付を計算
	now := time.Now()
	endDate := now.Format("2006-01-02")
	startDate := now.AddDate(0, -1, 0).Format("2006-01-02")

	log.Printf("Testing aggregation for period: %s to %s", startDate, endDate)

	// まずは全車両のサマリーを取得
	log.Println("\n=== Getting vehicle monthly summary ===")
	summaryResp, err := aggService.GetVehicleMonthlySummary(context.Background(), &pb.GetVehicleMonthlySummaryRequest{
		StartDate: startDate,
		EndDate:   endDate,
	})
	if err != nil {
		log.Fatalf("Failed to get vehicle summary: %v", err)
	}

	log.Printf("Total vehicles: %d", summaryResp.TotalVehicles)
	log.Printf("Period: %s", summaryResp.Period)

	// 最初の1車両のデータを詳細表示
	if len(summaryResp.VehicleSummaries) > 0 {
		firstVehicle := summaryResp.VehicleSummaries[0]
		log.Printf("\n=== First Vehicle: %s ===", firstVehicle.CarCc)
		log.Printf("Number of months: %d", len(firstVehicle.Summaries))

		for _, s := range firstVehicle.Summaries {
			log.Printf("  %s: 距離=%.1fkm, 給油=%.1fL, 運行=%d回, 燃費=%.2fkm/L",
				s.YearMonth, s.TotalDistance, s.TotalFuel, s.TripCount, s.AvgFuelEfficiency)
		}

			// 月次データから直接CSV生成（再取得しない）
		log.Printf("\n=== CSV Export (from cached data) ===")
		csvData := "年月,車両CC,走行距離(km),給油量(L),運行回数,平均燃費(km/L)\n"
		for _, s := range firstVehicle.Summaries {
			csvData += fmt.Sprintf("%s,%s,%.1f,%.1f,%d,%.2f\n",
				s.YearMonth, s.CarCc, s.TotalDistance, s.TotalFuel, s.TripCount, s.AvgFuelEfficiency)
		}
		log.Printf("CSV Data:\n%s", csvData)
	} else {
		log.Println("No vehicles found in the specified period")
	}
}
