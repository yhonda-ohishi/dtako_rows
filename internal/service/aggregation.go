package service

import (
	"context"
	"fmt"
	"log"
	"sort"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MonthlyFuelSummary 月次給油量サマリー
type MonthlyFuelSummary struct {
	CarCC         string  // 車輌CC
	YearMonth     string  // 年月 (YYYY-MM形式)
	TotalDistance float64 // 総走行距離
	TotalFuel     float64 // 総給油量（計算値: 走行距離 / 燃費）
	TripCount     int32   // 運行回数
}

// GetMonthlyFuelConsumption 車両ごとの月次給油量を集計
//
// 指定期間の運行データから、車両ごと・月ごとの給油量を集計します。
// 給油量は走行距離から推定計算します（実際の給油データがない場合）。
func (s *DtakoRowsService) GetMonthlyFuelConsumption(ctx context.Context, carCC string, startDate, endDate string) ([]*MonthlyFuelSummary, error) {
	log.Printf("GetMonthlyFuelConsumption: car_cc=%s, start=%s, end=%s", carCC, startDate, endDate)

	// バリデーション
	if carCC == "" {
		return nil, status.Error(codes.InvalidArgument, "car_cc is required")
	}

	// 新しいフィルタリングメソッドを使用
	allRows, err := s.ListByCarCCAndDateRange(ctx, carCC, startDate, endDate, 0)
	if err != nil {
		log.Printf("Failed to list rows with filter: %v", err)
		return nil, err
	}

	log.Printf("Filtered %d rows for car_cc=%s", len(allRows), carCC)

	// 月次集計
	monthlyData := make(map[string]*MonthlyFuelSummary)

	for _, row := range allRows {
		opDate, err := time.Parse(time.RFC3339, row.OperationDate)
		if err != nil {
			continue
		}

		yearMonth := opDate.Format("2006-01")

		if _, exists := monthlyData[yearMonth]; !exists {
			monthlyData[yearMonth] = &MonthlyFuelSummary{
				CarCC:     row.CarCc,
				YearMonth: yearMonth,
			}
		}

		summary := monthlyData[yearMonth]
		summary.TotalDistance += row.TotalDistance
		summary.TripCount++

		// 燃費を10km/Lと仮定して給油量を計算（実際の燃費データがあればそれを使用）
		// TODO: 車両マスタから実際の燃費を取得
		const averageFuelEfficiency = 10.0 // km/L
		summary.TotalFuel = summary.TotalDistance / averageFuelEfficiency
	}

	// マップを配列に変換してソート
	results := make([]*MonthlyFuelSummary, 0, len(monthlyData))
	for _, summary := range monthlyData {
		results = append(results, summary)
	}

	// 年月でソート
	sort.Slice(results, func(i, j int) bool {
		return results[i].YearMonth < results[j].YearMonth
	})

	log.Printf("Aggregated %d months of data", len(results))
	return results, nil
}

// GetVehicleMonthlySummary 全車両の月次サマリーを取得
//
// 指定期間の全車両の月次走行距離・給油量を集計します。
func (s *DtakoRowsService) GetVehicleMonthlySummary(ctx context.Context, startDate, endDate string) (map[string][]*MonthlyFuelSummary, error) {
	log.Printf("GetVehicleMonthlySummary: start=%s, end=%s", startDate, endDate)

	// 新しいフィルタリングメソッドを使用（日付範囲のみ）
	allRows, err := s.ListByDateRange(ctx, startDate, endDate, 0)
	if err != nil {
		log.Printf("Failed to list rows with filter: %v", err)
		return nil, err
	}

	log.Printf("Processing %d rows for vehicle summary", len(allRows))

	// 車両ごと・月次集計
	vehicleMonthlyData := make(map[string]map[string]*MonthlyFuelSummary)

	for _, row := range allRows {
		opDate, err := time.Parse(time.RFC3339, row.OperationDate)
		if err != nil {
			continue
		}

		yearMonth := opDate.Format("2006-01")
		carCC := row.CarCc

		if _, exists := vehicleMonthlyData[carCC]; !exists {
			vehicleMonthlyData[carCC] = make(map[string]*MonthlyFuelSummary)
		}

		if _, exists := vehicleMonthlyData[carCC][yearMonth]; !exists {
			vehicleMonthlyData[carCC][yearMonth] = &MonthlyFuelSummary{
				CarCC:     carCC,
				YearMonth: yearMonth,
			}
		}

		summary := vehicleMonthlyData[carCC][yearMonth]
		summary.TotalDistance += row.TotalDistance
		summary.TripCount++

		const averageFuelEfficiency = 10.0
		summary.TotalFuel = summary.TotalDistance / averageFuelEfficiency
	}

	// マップを整形
	results := make(map[string][]*MonthlyFuelSummary)
	for carCC, monthlyData := range vehicleMonthlyData {
		summaries := make([]*MonthlyFuelSummary, 0, len(monthlyData))
		for _, summary := range monthlyData {
			summaries = append(summaries, summary)
		}
		// 年月でソート
		sort.Slice(summaries, func(i, j int) bool {
			return summaries[i].YearMonth < summaries[j].YearMonth
		})
		results[carCC] = summaries
	}

	log.Printf("Aggregated data for %d vehicles", len(results))
	return results, nil
}

// PrintMonthlySummary 月次サマリーをログ出力（デバッグ用）
func PrintMonthlySummary(summaries []*MonthlyFuelSummary) {
	log.Println("=== Monthly Fuel Summary ===")
	for _, s := range summaries {
		log.Printf("%s | 車両: %s | 走行距離: %.1fkm | 給油量: %.1fL | 運行回数: %d回",
			s.YearMonth, s.CarCC, s.TotalDistance, s.TotalFuel, s.TripCount)
	}
}

// GetDailySummary 日次サマリーを取得
//
// 指定車両の日次走行距離・給油量を集計します。
func (s *DtakoRowsService) GetDailySummary(ctx context.Context, carCC string, startDate, endDate string) (map[string]*MonthlyFuelSummary, error) {
	log.Printf("GetDailySummary: car_cc=%s, start=%s, end=%s", carCC, startDate, endDate)

	if carCC == "" {
		return nil, status.Error(codes.InvalidArgument, "car_cc is required")
	}

	// 新しいフィルタリングメソッドを使用
	allRows, err := s.ListByCarCCAndDateRange(ctx, carCC, startDate, endDate, 0)
	if err != nil {
		return nil, err
	}

	dailyData := make(map[string]*MonthlyFuelSummary)

	for _, row := range allRows {
		opDate, err := time.Parse(time.RFC3339, row.OperationDate)
		if err != nil {
			continue
		}

		dateKey := opDate.Format("2006-01-02")

		if _, exists := dailyData[dateKey]; !exists {
			dailyData[dateKey] = &MonthlyFuelSummary{
				CarCC:     row.CarCc,
				YearMonth: dateKey, // 日付を格納
			}
		}

		summary := dailyData[dateKey]
		summary.TotalDistance += row.TotalDistance
		summary.TripCount++

		const averageFuelEfficiency = 10.0
		summary.TotalFuel = summary.TotalDistance / averageFuelEfficiency
	}

	log.Printf("Aggregated %d days of data", len(dailyData))
	return dailyData, nil
}

// FormatSummaryAsCSV 集計結果をCSV形式で出力（エクスポート用）
func FormatSummaryAsCSV(summaries []*MonthlyFuelSummary) string {
	csv := "年月,車両CC,走行距離(km),給油量(L),運行回数\n"
	for _, s := range summaries {
		csv += fmt.Sprintf("%s,%s,%.1f,%.1f,%d\n",
			s.YearMonth, s.CarCC, s.TotalDistance, s.TotalFuel, s.TripCount)
	}
	return csv
}
