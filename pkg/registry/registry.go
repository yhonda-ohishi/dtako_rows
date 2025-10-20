package registry

import (
	"log"

	"github.com/yhonda-ohishi/dtako_rows/internal/repository"
	"github.com/yhonda-ohishi/dtako_rows/internal/service"
	pb "github.com/yhonda-ohishi/dtako_rows/proto"
	"google.golang.org/grpc"
	"gorm.io/gorm"
)

// Register dtako_rowsサービスをgRPCサーバーに登録
// 外部から呼び出し可能な公開API
func Register(grpcServer *grpc.Server, db *gorm.DB) {
	// リポジトリ初期化
	rowRepo := repository.NewDtakoRowRepository(db)

	// サービス初期化
	dtakoRowsService := service.NewDtakoRowsService(rowRepo)

	// サービス登録
	pb.RegisterDtakoRowsServiceServer(grpcServer, dtakoRowsService)

	log.Println("✓ DtakoRowsService registered")
}
