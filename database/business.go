package database

import (
	"errors"
	"github.com/ethereum/go-ethereum/log"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Business 代表业务系统注册的信息。
type Business struct {
	GUID        uuid.UUID `gorm:"primaryKey" json:"guid"`
	BusinessUid string    `json:"business_uid"`
	NotifyUrl   string    `json:"notify_url"`
	Timestamp   uint64
}

// BusinessDB 定义了对 business 表的写操作接口（包含读接口 BusinessView）。
type BusinessDB interface {
	BusinessView

	StoreBusiness(*Business) error
}

// businessDB 是 BusinessDB 的具体实现。
type businessDB struct {
	gorm *gorm.DB
}

// BusinessView 定义了对 business 表的读操作接口。
type BusinessView interface {
	QueryBusinessList() ([]*Business, error)
	QueryBusinessByUuid(string) (*Business, error)
}

// NewBusinessDB 创建一个 BusinessDB 实例，供上层依赖注入使用。
func NewBusinessDB(db *gorm.DB) BusinessDB {
	return &businessDB{gorm: db}
}

// QueryBusinessList 查询 business 表中的所有业务记录。
func (db *businessDB) QueryBusinessList() ([]*Business, error) {
	var business []*Business
	err := db.gorm.Table("business").Find(&business).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, err
	}
	return business, err
}

// QueryBusinessByUuid 根据 business_uid 字段查询某个业务记录。
func (db *businessDB) QueryBusinessByUuid(businessUuid string) (*Business, error) {
	var business *Business
	result := db.gorm.Table("business").Where("business_uid", businessUuid).First(&business)
	if result.Error != nil {
		log.Error("query business all fail", "Err", result.Error)
		return nil, result.Error
	}
	return business, nil
}

// StoreBusiness 将新的业务记录插入 business 表。
func (db *businessDB) StoreBusiness(business *Business) error {
	result := db.gorm.Table("business").Create(business)
	return result.Error
}
