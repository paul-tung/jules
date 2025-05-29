package main

import (
	"auth-service/config"
	"auth-service/controllers"
	"auth-service/middlewares"
	"auth-service/utils"
	"log"
	"net/http" // Required for http.StatusOK

	"github.com/gin-gonic/gin"
)

func main() {
	// 1. Load configuration
	appConfig := config.LoadConfig()

	// 2. Initialize database connection
	utils.InitDB()      // InitDB internally uses config.LoadConfig()
	db := utils.GetDB() // Get DB instance after InitDB
	defer utils.DisconnectDB()

	// 3. Ensure database indexes
	// User indexes (assuming CreateIndexes in utils/db.go handles users collection)
	if err := utils.CreateIndexes(db.Client()); err != nil {
		log.Fatalf("Error creating user indexes: %v", err)
	}
	// Folder indexes
	if err := controllers.EnsureFolderIndexes(db); err != nil {
		log.Fatalf("Error creating folder indexes: %v", err)
	}

	// 4. Initialize Gin router
	router := gin.Default()
	// Increase default multipart memory limit (default is 32 MiB)
	router.MaxMultipartMemory = 64 << 20 // 64 MiB

	// 5. Initialize controllers
	authController := controllers.NewAuthController(db)
	videoController := controllers.NewVideoController(db, appConfig)
	folderController := controllers.NewFolderController(db)

	// 6. Define Routes

	// Public routes (authentication)
	authRoutes := router.Group("/api/auth")
	{
		authRoutes.POST("/register", authController.Register)
		authRoutes.POST("/login", authController.Login)
	}

	// Video routes (protected)
	videoRoutes := router.Group("/api/videos")
	videoRoutes.Use(middlewares.AuthMiddleware())
	{
		videoRoutes.POST("/upload", videoController.UploadVideo)
		videoRoutes.GET("", videoController.ListUserVideos)
		videoRoutes.GET("/:videoId", videoController.GetVideoByID)
		videoRoutes.PUT("/:videoId", videoController.UpdateVideoMetadata)
		videoRoutes.DELETE("/:videoId", videoController.DeleteVideo)
		videoRoutes.PUT("/:videoId/folder", videoController.AssignVideoToFolder) // Assign/unassign video to/from folder
	}

	// Folder routes (protected)
	folderRoutes := router.Group("/api/folders")
	folderRoutes.Use(middlewares.AuthMiddleware())
	{
		folderRoutes.POST("", folderController.CreateFolder)
		folderRoutes.GET("", folderController.ListFolders)
		folderRoutes.PUT("/:folderId", folderController.UpdateFolder)
		folderRoutes.DELETE("/:folderId", folderController.DeleteFolder)
	}

	// Example protected profile route
	userProfileRoutes := router.Group("/api/user")
	userProfileRoutes.Use(middlewares.AuthMiddleware())
	{
		userProfileRoutes.GET("/profile", func(c *gin.Context) {
			userID := c.GetString("userID")
			email := c.GetString("email")
			c.JSON(http.StatusOK, gin.H{
				"message": "This is a protected user profile route",
				"userID":  userID,
				"email":   email,
			})
		})
	}

	// 7. Start server
	log.Println("Starting server on port 8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatal("Failed to start server: ", err)
	}
}
