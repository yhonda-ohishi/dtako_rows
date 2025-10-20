package models

import (
	"time"
)

// DtakoRow 運行データ (prod_dbのdtako_rowsテーブル)
type DtakoRow struct {
	ID                   string     `gorm:"primaryKey;column:id"`
	UnkoNo               string     `gorm:"column:運行NO;index"`
	SharyouCC            string     `gorm:"column:車輌CC;index"`
	JomuinCD1            string     `gorm:"column:乗務員CD1;index"`
	ShukkoDateTime       time.Time  `gorm:"column:出庫日時;index"`
	KikoDateTime         *time.Time `gorm:"column:帰庫日時"`
	UnkoDate             time.Time  `gorm:"column:運行日;index"`
	TaishouJomuinKubun   int        `gorm:"column:対象乗務員区分"`
	SoukouKyori          float64    `gorm:"column:走行距離"`
	NenryouShiyou        float64    `gorm:"column:燃料使用量"`
	Created              time.Time  `gorm:"column:created"`
	Modified             time.Time  `gorm:"column:modified"`
}

// TableName テーブル名を指定
func (DtakoRow) TableName() string {
	return "dtako_rows"
}
