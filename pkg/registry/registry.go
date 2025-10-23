package registry

import (
	"log"

	dbgrpc "buf.build/gen/go/yhonda-ohishi/db-service/grpc/go/_gogrpc"
	"github.com/yhonda-ohishi/dtako_rows/v3/internal/service"
	pb "github.com/yhonda-ohishi/dtako_rows/v3/proto"
	"google.golang.org/grpc"
)

// Register dtako_rowsサービスをgRPCサーバーに登録
//
// desktop-serverから呼び出され、単一プロセス内でサービス登録を行う。
// このパターンにより、複数のサービスを1つのプロセスで管理できる。
//
// 登録されるサービス:
//   - DTakoRowsService: 運行データ管理（db_serviceへのプロキシ）
//
// データアクセス:
//   - db_service経由で行う（同一プロセス内gRPC呼び出し）
//   - db_serviceがDB操作を担当し、このサービスは透過的にプロキシする
//
// 使い方（desktop-server内）:
//   dtakoRowsClient := dbgrpc.NewDb_DTakoRowsServiceClient(localConn)  // 同一プロセス内接続
//   dtako_rows_registry.RegisterWithClient(grpcServer, dtakoRowsClient)
func Register(grpcServer *grpc.Server, dbServiceAddr string) error {
	log.Println("Registering dtako_rows service...")

	// db_serviceへのプロキシサービスを登録（外部接続）
	svc, err := service.NewDtakoRowsService(dbServiceAddr)
	if err != nil {
		log.Printf("Failed to create dtako_rows service: %v", err)
		return err
	}
	dbgrpc.RegisterDb_DTakoRowsServiceServer(grpcServer, svc)

	log.Println("dtako_rows service registered successfully")
	return nil
}

// RegisterWithClient 既存のdb_serviceクライアントを使ってサービスを登録（desktop-server統合用）
//
// desktop-server内で同一プロセスのdb_serviceに接続する場合に使用
func RegisterWithClient(grpcServer *grpc.Server, dbClient dbgrpc.Db_DTakoRowsServiceClient) {
	log.Println("Registering dtako_rows services with existing db_service client...")

	// 既存クライアントを使ってサービスを作成
	svc := service.NewDtakoRowsServiceWithClient(dbClient)
	dbgrpc.RegisterDb_DTakoRowsServiceServer(grpcServer, svc)

	// 集計サービスも登録
	aggSvc := service.NewDtakoRowsAggregationServiceWithClient(dbClient)
	pb.RegisterDtakoRowsServiceServer(grpcServer, aggSvc)

	log.Println("dtako_rows services registered successfully (DTakoRowsService + DtakoRowsService)")
}
