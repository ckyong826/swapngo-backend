package main

import (
	"log"
	"net/http"
	"os"

	"github.com/ckyong826/swapngo-backend/pkg/database"
	"github.com/ckyong826/swapngo-backend/pkg/response"
	"github.com/gin-gonic/gin"
)

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	// 1. DB connection: defaults match docker-compose.yml (Postgres published on host 5433).
	_, err := database.InitDB(
		getenv("DB_HOST", "localhost"),
		getenv("DB_USER", "root"),
		getenv("DB_PASSWORD", "secretpassword"),
		getenv("DB_NAME", "swapngo"),
		getenv("DB_PORT", "5433"),
	)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// TODO: capture *gorm.DB from InitDB for AutoMigrate and queries

	// 2. 初始化 Gin 引擎
	router := gin.Default()

	// 3. 测试我们的统一响应包
	router.GET("/health", func(c *gin.Context) {
		// 模拟从数据库获取到的一些状态数据
		serverStatus := map[string]string{
			"db_status": "connected",
			"version":   "1.0.0",
		}
		
		// 使用 pkg/response 优雅地返回数据
		response.Success(c, serverStatus)
	})

	router.GET("/error-test", func(c *gin.Context) {
		// 模拟一个错误返回
		response.Error(c, http.StatusBadRequest, "Invalid request parameters")
	})

	// 4. 启动服务器
	log.Println("Server is starting on port 8080...")
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}