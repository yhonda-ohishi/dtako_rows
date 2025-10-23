package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	dbgrpc "buf.build/gen/go/yhonda-ohishi/db-service/grpc/go/_gogrpc"
	"github.com/joho/godotenv"
	"github.com/yhonda-ohishi/dtako_rows/v3/internal/service"
	pb "github.com/yhonda-ohishi/dtako_rows/v3/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// .envファイルの読み込み
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using environment variables")
	}

	// db_serviceアドレス設定
	dbServiceAddr := os.Getenv("DB_SERVICE_ADDR")
	if dbServiceAddr == "" {
		dbServiceAddr = "localhost:50051"
	}

	// サービス初期化（db_service経由でデータアクセス）
	dtakoRowsService, err := service.NewDtakoRowsService(dbServiceAddr)
	if err != nil {
		log.Fatalf("Failed to create service: %v", err)
	}

	// 集計サービス初期化
	aggregationService, err := service.NewDtakoRowsAggregationService(dbServiceAddr)
	if err != nil {
		log.Fatalf("Failed to create aggregation service: %v", err)
	}

	// gRPCサーバー作成
	grpcServer := grpc.NewServer()

	// サービス登録
	dbgrpc.RegisterDb_DTakoRowsServiceServer(grpcServer, dtakoRowsService)
	pb.RegisterDtakoRowsServiceServer(grpcServer, aggregationService)

	// リフレクション登録（grpcurlなどのツール用）
	reflection.Register(grpcServer)

	// ポート設定
	port := os.Getenv("GRPC_PORT")
	if port == "" {
		port = "50053"
	}

	// リスナー作成
	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// シグナルハンドリング（Graceful Shutdown）
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received shutdown signal, stopping server...")
		grpcServer.GracefulStop()
	}()

	// サーバー起動
	log.Printf("Starting gRPC server on port %s...", port)
	log.Printf("Services registered:")
	log.Printf("  - DTakoRowsService (proxy to db_service at %s)", dbServiceAddr)
	log.Printf("  - DtakoRowsService (aggregation logic)")

	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
