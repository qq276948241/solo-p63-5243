package main

import (
	"clinic/internal/appointment"
	"clinic/internal/common"
	"clinic/internal/schedule"
	"clinic/internal/user"
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	if err := common.LoadConfig(); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if err := common.InitDB(); err != nil {
		log.Fatalf("Failed to init database: %v", err)
	}

	if err := common.AutoMigrate(
		&user.User{},
		&user.DoctorProfile{},
		&schedule.Schedule{},
		&appointment.Appointment{},
	); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	userService := user.NewService(common.DB)
	scheduleService := schedule.NewService(common.DB)
	appointmentService := appointment.NewService(common.DB, scheduleService)

	userHandler := user.NewHandler(userService)
	scheduleHandler := schedule.NewHandler(scheduleService)
	appointmentHandler := appointment.NewHandler(appointmentService)

	r := gin.Default()

	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	api := r.Group("/api")
	{
		userHandler.RegisterRoutes(api)
		scheduleHandler.RegisterRoutes(api)
		appointmentHandler.RegisterRoutes(api)
	}

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})

	log.Printf("Server starting on port %s", common.AppConfig.ServerPort)
	if err := r.Run(":" + common.AppConfig.ServerPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
