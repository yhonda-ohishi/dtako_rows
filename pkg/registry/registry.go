package registry

import (
	"context"
	"log"

	dbpb "github.com/yhonda-ohishi/db_service/src/proto"
	"github.com/yhonda-ohishi/dtako_rows/v3/internal/service"
	pb "github.com/yhonda-ohishi/dtako_rows/v3/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Register dtako_rowsサービスをgRPCサーバーに登録
//
// 使い方:
//   1. Standalone モード: Register(grpcServer)
//      - 外部の db_service (localhost:50051) に接続
//      - Db_DTakoRowsService と DtakoRowsService の両方を登録
//
//   2. Desktop-server 統合モード: Register(grpcServer, dbServer)
//      - 同一プロセス内の db_service サーバー実装を使用
//      - DtakoRowsService のみ登録（Db_DTakoRowsService は重複回避のため登録しない）
//
// パラメータ:
//   - grpcServer: gRPCサーバーインスタンス
//   - dbServer: (オプショナル) 同一プロセス内の db_service サーバー実装
func Register(grpcServer *grpc.Server, dbServer ...dbpb.Db_DTakoRowsServiceServer) error {
	// Desktop-server統合モード: dbServerが渡された場合
	if len(dbServer) > 0 && dbServer[0] != nil {
		log.Println("Registering dtako_rows in desktop-server integration mode...")
		RegisterWithServer(grpcServer, dbServer[0])
		return nil
	}

	// Standaloneモード: 外部db_serviceに接続
	log.Println("Registering dtako_rows in standalone mode...")

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

	log.Println("dtako_rows services registered successfully (Db_DTakoRowsService + DtakoRowsService)")
}

// RegisterWithServer 既存のdb_serviceサーバー実装を使ってサービスを登録
//
// 同一プロセス内でdb_serviceとdtako_rowsを統合する場合に使用。
// desktop-server側でアダプター実装が不要になります。
//
// 注意: この関数は DtakoRowsService のみを登録します。
// Db_DTakoRowsService は desktop-server 側で既に登録されているため、
// 重複登録を避けるためにここでは登録しません。
func RegisterWithServer(grpcServer *grpc.Server, dbServer dbpb.Db_DTakoRowsServiceServer) {
	log.Println("Registering DtakoRowsService (aggregation + proxy) with existing db_service server...")

	// サーバー実装をクライアントインターフェースとしてラップ
	client := &localServerClient{server: dbServer}

	// DtakoRowsService のみ登録（Db_DTakoRowsService は登録しない）
	aggSvc := service.NewDtakoRowsAggregationServiceWithClient(client)
	pb.RegisterDtakoRowsServiceServer(grpcServer, aggSvc)

	log.Println("DtakoRowsService registered successfully")
}

// localServerClient はサーバー実装をクライアントインターフェースに適合させるアダプター
//
// 同一プロセス内でサーバーメソッドを直接呼び出すことで、
// ネットワーク経由の gRPC 呼び出しをバイパスします。
type localServerClient struct {
	server dbpb.Db_DTakoRowsServiceServer
}

func (c *localServerClient) Get(ctx context.Context, req *dbpb.Db_GetDTakoRowsRequest, opts ...grpc.CallOption) (*dbpb.Db_DTakoRowsResponse, error) {
	return c.server.Get(ctx, req)
}

func (c *localServerClient) List(ctx context.Context, req *dbpb.Db_ListDTakoRowsRequest, opts ...grpc.CallOption) (*dbpb.Db_ListDTakoRowsResponse, error) {
	return c.server.List(ctx, req)
}

func (c *localServerClient) GetByOperationNo(ctx context.Context, req *dbpb.Db_GetDTakoRowsByOperationNoRequest, opts ...grpc.CallOption) (*dbpb.Db_ListDTakoRowsResponse, error) {
	return c.server.GetByOperationNo(ctx, req)
}
