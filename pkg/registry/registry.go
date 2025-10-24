package registry

import (
	"log"

	dbpb "github.com/yhonda-ohishi/db_service/src/proto"
	"github.com/yhonda-ohishi/dtako_rows/v3/internal/service"
	pb "github.com/yhonda-ohishi/dtako_rows/v3/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Register dtako_rowsサービスをgRPCサーバーに登録
//
// desktop-serverから呼び出され、単一プロセス内でサービス登録を行う。
// このパターンにより、複数のサービスを1つのプロセスで管理できる。
//
// 登録されるサービス:
//   - Db_DTakoRowsService: 運行データCRUDプロキシ（db_serviceへ）
//   - DtakoRowsService: 集計機能 + GetRow/ListRowsプロキシ
//
// データアクセス:
//   - db_service経由で行う（同一プロセス内gRPC呼び出し）
//   - db_serviceがDB操作を担当し、このサービスは透過的にプロキシする
func Register(grpcServer *grpc.Server) error {
	log.Println("Registering dtako_rows services...")

	// Create db_service client
	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("Failed to create db_service client: %v", err)
		return err
	}

	dbClient := dbpb.NewDb_DTakoRowsServiceClient(conn)

	// Register both services using RegisterWithClient
	RegisterWithClient(grpcServer, dbClient)

	log.Println("dtako_rows services registered successfully (Db_DTakoRowsService + DtakoRowsService)")
	return nil
}

// RegisterWithClient 既存のdb_serviceクライアントを使ってサービスを登録（desktop-server統合用）
//
// desktop-server内で同一プロセスのdb_serviceに接続する場合に使用
func RegisterWithClient(grpcServer *grpc.Server, dbClient dbpb.Db_DTakoRowsServiceClient) {
	log.Println("Registering dtako_rows services with existing db_service client...")

	// 既存クライアントを使ってサービスを作成
	svc := service.NewDtakoRowsServiceWithClient(dbClient)
	dbpb.RegisterDb_DTakoRowsServiceServer(grpcServer, svc)

	// 集計サービスも登録
	aggSvc := service.NewDtakoRowsAggregationServiceWithClient(dbClient)
	pb.RegisterDtakoRowsServiceServer(grpcServer, aggSvc)

	log.Println("dtako_rows services registered successfully (DTakoRowsService + DtakoRowsService)")
}
