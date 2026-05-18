package mysqlbackup

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// Ping 检查 MySQL 实例是否存活（对齐 mysql_golang_tools mysqlping）。
func Ping(ctx context.Context, host string, port int, user, password string) error {
	if port <= 0 {
		port = 3306
	}
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/mysql?timeout=5s&readTimeout=5s&writeTimeout=5s",
		user, password, host, port)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	pctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	return db.PingContext(pctx)
}
