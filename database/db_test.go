package database

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"exchange-wallet-service/config"
)

/*链接数据库（必须先创建数据库）*/
func TestNewDB(t *testing.T) {
	dbCfg := config.DBConfig{
		Host: "localhost",
		Port: 5432,
		Name: "test_db",
		User: "steven_shaw",
	}

	db, err := NewDB(context.Background(), dbCfg)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()
}

/*表迁移（必须先创建数据库）*/
func TestExecuteSQLMigration(t *testing.T) {
	dbCfg := config.DBConfig{
		Host: "localhost",
		Port: 5432,
		Name: "test_db",
		User: "steven_shaw",
	}

	db, err := NewDB(context.Background(), dbCfg)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	// 准备测试 SQL 目录和文件
	testDir := "./test_migrations"
	testSQL := "CREATE TABLE IF NOT EXISTS test_table (id SERIAL PRIMARY KEY);"
	err = os.MkdirAll(testDir, 0755)
	if err != nil {
		t.Fatalf("failed to create test dir: %v", err)
	}
	defer os.RemoveAll(testDir) // 清理

	testFile := filepath.Join(testDir, "001_create_test_table.sql")
	err = os.WriteFile(testFile, []byte(testSQL), 0644)
	if err != nil {
		t.Fatalf("failed to write test SQL file: %v", err)
	}

	// 执行迁移
	err = db.ExecuteSQLMigration(testDir)
	if err != nil {
		t.Errorf("ExecuteSQLMigration failed: %v", err)
	}
}
