package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/yhonda-ohishi/dtako_rows/v2/internal/service"
	pb "github.com/yhonda-ohishi/dtako_rows/v2/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// .envファイルの読み込み
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using environment variables")
	}

	// サービス初期化（db_service経由でデータアクセス）
	dtakoRowsService := service.NewDtakoRowsService()

	// gRPCサーバー作成
	grpcServer := grpc.NewServer()

	// サービス登録
	pb.RegisterDtakoRowsServiceServer(grpcServer, dtakoRowsService)

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
	log.Printf("  - DtakoRowsService (data access via db_service)")
	log.Printf("Note: Requires db_service running on localhost:50051")

	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
