package database

import (
	"fmt"
	"log"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// InitDB 负责建立并返回 PostgreSQL 的连接
func InitDB(host, user, password, dbname, port string) (*gorm.DB, error) {
	// 拼接 DSN (Data Source Name) 数据源字符串
	// 根据我们的 docker-compose.yml 设定，时区设为马来西亚吉隆坡时间
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Kuala_Lumpur",
		host, user, password, dbname, port)

	// 使用 GORM 打开连接，并配置日志记录器（方便在终端查看生成的 SQL 语句）
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Printf("Failed to connect to database: %v\n", err)
		return nil, err
	}

	// 获取底层的 sql.DB 对象以便设置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// 配置企业级连接池机制，避免并发量大时连接数耗尽
	sqlDB.SetMaxIdleConns(10)           // 设置空闲连接池中连接的最大数量
	sqlDB.SetMaxOpenConns(100)          // 设置打开数据库连接的最大数量
	sqlDB.SetConnMaxLifetime(time.Hour) // 设置连接可复用的最大时间

	log.Println("PostgreSQL connection successfully established!")
	return db, nil
}