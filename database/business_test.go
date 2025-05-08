package database

import (
	"database/sql/driver"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)

	dialector := postgres.New(postgres.Config{
		Conn: db,
	})
	gormDB, err := gorm.Open(dialector, &gorm.Config{})
	assert.NoError(t, err)

	return gormDB, mock
}

func TestQueryBusinessList(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	defer func() {
		db, _ := gormDB.DB()
		db.Close()
	}()

	mock.ExpectQuery(`SELECT \* FROM "business"`).
		WillReturnRows(sqlmock.NewRows([]string{"guid", "business_uid", "notify_url", "timestamp"}).
			AddRow(uuid.New(), "biz-1", "http://callback", 123456))

	db := NewBusinessDB(gormDB)
	bizList, err := db.QueryBusinessList()

	assert.NoError(t, err)
	assert.Len(t, bizList, 1)
	assert.Equal(t, "biz-1", bizList[0].BusinessUid)
}

func TestQueryBusinessByUuid(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	defer func() {
		db, _ := gormDB.DB()
		db.Close()
	}()

	mock.ExpectQuery(`SELECT \* FROM "business" WHERE "business_uid" = \$1 ORDER BY "business"\."guid" LIMIT \$2`).
		WithArgs("biz-uuid", 1).
		WillReturnRows(sqlmock.NewRows([]string{"guid", "business_uid", "notify_url", "timestamp"}).
			AddRow(uuid.New(), "biz-uuid", "http://callback", 654321))

	db := NewBusinessDB(gormDB)
	biz, err := db.QueryBusinessByUuid("biz-uuid")

	assert.NoError(t, err)
	assert.Equal(t, "biz-uuid", biz.BusinessUid)
}

func TestStoreBusiness(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	defer func() {
		db, _ := gormDB.DB()
		db.Close()
	}()

	biz := &Business{
		GUID:        uuid.New(),
		BusinessUid: "biz-123",
		NotifyUrl:   "http://notify.me",
		Timestamp:   111111,
	}

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "business"`).
		WithArgs(biz.GUID, biz.BusinessUid, biz.NotifyUrl, biz.Timestamp).
		WillReturnResult(driver.ResultNoRows)
	mock.ExpectCommit()

	db := NewBusinessDB(gormDB)
	err := db.StoreBusiness(biz)
	assert.NoError(t, err)
}
