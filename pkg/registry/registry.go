package registry

import (
	"github.com/yhonda-ohishi/dtako_rows/pkg/server"
	"google.golang.org/grpc"
)

// Register dtako_rowsサービスをgRPCサーバーに登録
// 外部から呼び出し可能な公開API
// DB接続は内部で管理され、環境変数から設定を読み込みます
func Register(grpcServer *grpc.Server) error {
	return server.Start(grpcServer)
}
