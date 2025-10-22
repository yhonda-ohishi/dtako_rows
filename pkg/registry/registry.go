package registry

import (
	"log"

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
//   - DtakoRowsService: 運行データ管理
//
// データアクセス:
//   - db_service経由で行う（同一プロセス内gRPC呼び出し）
//   - db_serviceがDB操作を担当し、このサービスはビジネスロジックのみ
func Register(grpcServer *grpc.Server) error {
	log.Println("Registering dtako_rows service...")

	// ビジネスロジックサービスのみ登録（DB接続不要）
	svc := service.NewDtakoRowsService()
	pb.RegisterDtakoRowsServiceServer(grpcServer, svc)

	log.Println("dtako_rows service registered successfully")
	return nil
}
