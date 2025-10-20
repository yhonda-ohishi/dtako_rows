package server

import (
	"log"
	"time"

	"github.com/yhonda-ohishi/dtako_rows/v2/internal/config"
	"github.com/yhonda-ohishi/dtako_rows/v2/internal/repository"
	"github.com/yhonda-ohishi/dtako_rows/v2/internal/service"
	pb "github.com/yhonda-ohishi/dtako_rows/v2/proto"
	"google.golang.org/grpc"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Start は既存のgRPCサーバーにdtako_rowsサービスを登録します
// 内部でDB接続を管理し、環境変数から設定を読み込みます
func Start(grpcServer *grpc.Server) error {
	// 環境変数から設定を読み込み
	cfg := config.LoadDatabaseConfig()

	// データベース接続
	db, err := connectDB(cfg)
	if err != nil {
		return err
	}

	// サービス登録
	repo := repository.NewDtakoRowRepository(db)
	svc := service.NewDtakoRowsService(repo)
	pb.RegisterDtakoRowsServiceServer(grpcServer, svc)

	log.Println("✓ DtakoRowsService registered")
	return nil
}

// connectDB データベース接続を作成
func connectDB(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	// ログレベル設定
	logLevel := logger.Info
	if log.Flags() != 0 {
		logLevel = logger.Info
	}

	db, err := gorm.Open(mysql.Open(cfg.DSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
		NowFunc: func() time.Time {
			return time.Now().Local()
		},
	})

	if err != nil {
		return nil, err
	}

	// コネクションプール設定
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return db, nil
}
