package strings

import (
	"regexp"
	"strings"
)

// PostgreSQL 保留字列表（示例，实际可以根据数据库的完整保留字列表）
var reservedWords = map[string]bool{
	"SELECT": true,
	"TABLE":  true,
	"INSERT": true,
	"DELETE": true,
	"UPDATE": true,
	"FROM":   true,
	"WHERE":  true,
	"GROUP":  true,
	"HAVING": true,
	"ORDER":  true,
	"BY":     true,
	"LIMIT":  true,
	"OFFSET": true,
	// 其他保留字...
}

// IsValidTableName 校验函数，检查表名是否符合规则
func IsValidTableName(tableName string) bool {
	// 1. 检查长度（以 PostgreSQL 为例，表名最大长度是 63）
	if len(tableName) == 0 || len(tableName) > 20 {
		return false
	}
	// 2. 只能包含字母、数字和下划线
	match, _ := regexp.MatchString(`^[a-zA-Z_][a-zA-Z0-9_]*$`, tableName)
	if !match {
		return false
	}
	// 3. 表名不能是数据库保留字
	if reservedWords[strings.ToUpper(tableName)] {
		return false
	}
	return true
}
