package repository

import (
	"context"
	"time"

	"github.com/yhonda-ohishi/dtako_rows/v2/internal/models"
	"gorm.io/gorm"
)

// DtakoRowRepository 運行データリポジトリ
type DtakoRowRepository interface {
	GetByID(ctx context.Context, id string) (*models.DtakoRow, error)
	List(ctx context.Context, page, pageSize int, orderBy string) ([]*models.DtakoRow, int64, error)
	Create(ctx context.Context, row *models.DtakoRow) error
	Update(ctx context.Context, row *models.DtakoRow) error
	Delete(ctx context.Context, id string) error
	Search(ctx context.Context, dateFrom, dateTo *time.Time, sharyouCC, jomuinCD1 string) ([]*models.DtakoRow, error)
}

type dtakoRowRepository struct {
	db *gorm.DB
}

// NewDtakoRowRepository リポジトリの作成
func NewDtakoRowRepository(db *gorm.DB) DtakoRowRepository {
	return &dtakoRowRepository{db: db}
}

// GetByID IDで運行データを取得
func (r *dtakoRowRepository) GetByID(ctx context.Context, id string) (*models.DtakoRow, error) {
	var row models.DtakoRow
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// List 運行データ一覧を取得
func (r *dtakoRowRepository) List(ctx context.Context, page, pageSize int, orderBy string) ([]*models.DtakoRow, int64, error) {
	var rows []*models.DtakoRow
	var total int64

	// デフォルトの並び順
	if orderBy == "" {
		orderBy = "出庫日時 DESC"
	}

	// 総件数取得
	if err := r.db.WithContext(ctx).Model(&models.DtakoRow{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// ページング
	offset := (page - 1) * pageSize
	if err := r.db.WithContext(ctx).
		Order(orderBy).
		Limit(pageSize).
		Offset(offset).
		Find(&rows).Error; err != nil {
		return nil, 0, err
	}

	return rows, total, nil
}

// Create 運行データを作成
func (r *dtakoRowRepository) Create(ctx context.Context, row *models.DtakoRow) error {
	return r.db.WithContext(ctx).Create(row).Error
}

// Update 運行データを更新
func (r *dtakoRowRepository) Update(ctx context.Context, row *models.DtakoRow) error {
	return r.db.WithContext(ctx).Save(row).Error
}

// Delete 運行データを削除
func (r *dtakoRowRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&models.DtakoRow{}).Error
}

// Search 条件で運行データを検索
func (r *dtakoRowRepository) Search(ctx context.Context, dateFrom, dateTo *time.Time, sharyouCC, jomuinCD1 string) ([]*models.DtakoRow, error) {
	var rows []*models.DtakoRow
	query := r.db.WithContext(ctx)

	// 日付範囲検索
	if dateFrom != nil {
		query = query.Where("出庫日時 >= ?", dateFrom)
	}
	if dateTo != nil {
		query = query.Where("出庫日時 <= ?", dateTo)
	}

	// 車輌CC検索
	if sharyouCC != "" {
		query = query.Where("車輌CC = ?", sharyouCC)
	}

	// 乗務員CD1検索
	if jomuinCD1 != "" {
		query = query.Where("乗務員CD1 = ?", jomuinCD1)
	}

	if err := query.Order("出庫日時 DESC").Find(&rows).Error; err != nil {
		return nil, err
	}

	return rows, nil
}
