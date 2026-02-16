package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"strings"

	useractivity "github.com/PayRam/activity-log/pkg/useractivity"
	"github.com/PayRam/activity-log/pkg/useractivity/ginmiddleware"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	dsn := strings.TrimSpace(os.Getenv("USER_ACTIVITY_POSTGRES_DSN"))
	if dsn == "" {
		log.Fatal("set USER_ACTIVITY_POSTGRES_DSN (for example: host=localhost user=postgres password=postgres dbname=postgres port=5432 sslmode=disable)")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("open db: %v", err)
	}

	client, err := useractivity.New(useractivity.Config{DB: db})
	if err != nil {
		log.Fatalf("init user activity client: %v", err)
	}

	if err := client.AutoMigrate(context.Background()); err != nil {
		log.Fatalf("auto-migrate: %v", err)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(ginmiddleware.Middleware(ginmiddleware.Config{
		Client:              client,
		CaptureRequestBody:  true,
		CaptureResponseBody: true,
		MaxBodyBytes:        1 * 1024 * 1024,
		Redact:              useractivity.RedactDefaultJSONKeys,
		ResponseRedact:      useractivity.RedactDefaultJSONKeys,
		SkipPaths:           []string{"/healthz"},
		CreateEnricher: func(c *gin.Context, req *useractivity.CreateRequest) {
			if v := c.GetHeader("X-Member-ID"); v != "" {
				if id, err := strconv.ParseUint(v, 10, 64); err == nil {
					memberID := uint(id)
					req.MemberID = &memberID
				}
			}
		},
		UpdateEnricher: func(c *gin.Context, req *useractivity.UpdateRequest, resp *ginmiddleware.CapturedResponse) {
			if resp.StatusCode >= 400 && req.Description == nil {
				msg := "request failed"
				req.Description = &msg
			}
		},
	}))

	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	router.POST("/echo", func(c *gin.Context) {
		var body map[string]any
		if err := c.BindJSON(&body); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, body)
	})

	log.Println("listening on :8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("run server: %v", err)
	}
}
