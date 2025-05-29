package controllers

import (
	"auth-service/config"
	"auth-service/models"
	"context"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"math"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// VideoController handles video-related requests.
type VideoController struct {
	DB         *mongo.Database
	AppConfig  *config.Config
}

// UpdateVideoRequest represents the request body for updating video metadata.
type UpdateVideoRequest struct {
	Title       *string  `json:"title"`
	Description *string  `json:"description"`
	Tags        []string `json:"tags"`
	Status      *string  `json:"status"` // e.g., "private", "public", "unlisted"
}

// AssignVideoToFolderRequest represents the request body for assigning a video to a folder.
// This struct is also defined in folder_controller.go, ensure consistency or move to a shared models/requests package.
type AssignVideoToFolderRequest struct {
	FolderID *string `json:"folderId"` // Optional hex string, nil means unassign
}


// NewVideoController creates a new VideoController.
func NewVideoController(db *mongo.Database, appConfig *config.Config) *VideoController {
	return &VideoController{DB: db, AppConfig: appConfig}
}

// Helper function to get userID from context
func getUserIDFromContext(c *gin.Context) (primitive.ObjectID, error) {
	userIDHex, exists := c.Get("userID")
	if !exists {
		return primitive.NilObjectID, fmt.Errorf("user ID not found in context")
	}
	userID, err := primitive.ObjectIDFromHex(userIDHex.(string))
	if err != nil {
		return primitive.NilObjectID, fmt.Errorf("invalid User ID format")
	}
	return userID, nil
}

// UploadVideo handles video file uploads.
func (vc *VideoController) UploadVideo(c *gin.Context) {
	// 1. Ensure upload path exists
	uploadDir := vc.AppConfig.VideoUploadPath
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create upload directory"})
		log.Printf("Error creating upload directory %s: %v\n", uploadDir, err)
		return
	}

	// 2. Get UserID from context (set by AuthMiddleware)
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// 3. Get the file from form data
	file, header, err := c.Request.FormFile("videoFile")
	if err != nil {
		if err == http.ErrMissingFile {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No video file provided in 'videoFile' field"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Error retrieving the file: %v", err)})
		log.Printf("Error retrieving file: %v\n", err)
		return
	}
	defer func(file multipart.File) {
		err := file.Close()
		if err != nil {
			log.Printf("Error closing uploaded file: %v\n", err)
		}
	}(file)

	// 4. Generate a unique filename
	originalFilename := header.Filename
	fileExtension := filepath.Ext(originalFilename)
	uniqueFilename := uuid.New().String() + fileExtension
	storagePath := filepath.Join(uploadDir, uniqueFilename) // Relative path for storage in DB if needed, but physical save uses full path

	// 5. Save the file to the filesystem
	outFile, err := os.Create(storagePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save video file"})
		log.Printf("Error creating file %s: %v\n", storagePath, err)
		return
	}
	defer func(outFile *os.File) {
		err := outFile.Close()
		if err != nil {
			log.Printf("Error closing output file %s: %v\n", storagePath, err)
		}
	}(outFile)

	_, err = io.Copy(outFile, file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to write video file to disk"})
		log.Printf("Error copying file to %s: %v\n", storagePath, err)
		return
	}

	// 6. Create video metadata in MongoDB
	video := models.Video{
		ID:               primitive.NewObjectID(),
		UserID:           userID,
		Title:            strings.TrimSuffix(originalFilename, fileExtension), // Default title to filename without extension
		OriginalFilename: originalFilename,
		StoragePath:      storagePath, // Store the full path or relative based on needs
		Status:           "uploaded",
		UploadedAt:       time.Now(),
		UpdatedAt:        time.Now(),
		// Optional fields can be added here or updated later
		Description:     c.PostForm("description"), // Example: get description from form
		Tags:            c.PostFormArray("tags"), // Example: get tags from form
	}

	collection := vc.DB.Collection("videos")
	_, err = collection.InsertOne(context.TODO(), video)
	if err != nil {
		// Attempt to remove the saved file if DB insert fails
		if removeErr := os.Remove(storagePath); removeErr != nil {
			log.Printf("Error removing file %s after DB insert failure: %v (original DB error: %v)\n", storagePath, removeErr, err)
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create video metadata in database"})
		log.Printf("Error inserting video metadata into DB: %v\n", err)
		return
	}

	// 7. Response
	c.JSON(http.StatusCreated, video)
}

// ListUserVideos handles listing videos for the authenticated user with pagination.
func (vc *VideoController) ListUserVideos(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	pageQuery := c.DefaultQuery("page", "1")
	pageSizeQuery := c.DefaultQuery("pageSize", "10")

	page, err := strconv.ParseInt(pageQuery, 10, 64)
	if err != nil || page < 1 {
		page = 1
	}
	pageSize, err := strconv.ParseInt(pageSizeQuery, 10, 64)
	if err != nil || pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 { // Max page size limit
		pageSize = 100
	}

	videosCollection := vc.DB.Collection("videos")
	filter := bson.M{"userId": userID}

	// Count total documents
	totalItems, err := videosCollection.CountDocuments(context.TODO(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count videos"})
		log.Printf("Error counting videos for user %s: %v\n", userID.Hex(), err)
		return
	}

	// Calculate total pages
	totalPages := int64(math.Ceil(float64(totalItems) / float64(pageSize)))

	// Find videos with pagination
	findOptions := options.Find()
	findOptions.SetSkip((page - 1) * pageSize)
	findOptions.SetLimit(pageSize)
	findOptions.SetSort(bson.D{{"uploadedAt", -1}}) // Sort by newest first

	cursor, err := videosCollection.Find(context.TODO(), filter, findOptions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve videos"})
		log.Printf("Error finding videos for user %s: %v\n", userID.Hex(), err)
		return
	}
	defer cursor.Close(context.TODO())

	var videos []models.Video
	if err = cursor.All(context.TODO(), &videos); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode videos"})
		log.Printf("Error decoding videos for user %s: %v\n", userID.Hex(), err)
		return
	}

	if videos == nil {
		videos = []models.Video{} // Return empty array instead of null
	}

	c.JSON(http.StatusOK, gin.H{
		"currentPage": page,
		"pageSize":    pageSize,
		"totalItems":  totalItems,
		"totalPages":  totalPages,
		"items":       videos,
	})
}

// GetVideoByID handles fetching a single video by its ID for the authenticated user.
func (vc *VideoController) GetVideoByID(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	videoIDHex := c.Param("videoId")
	videoID, err := primitive.ObjectIDFromHex(videoIDHex)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid video ID format"})
		return
	}

	var video models.Video
	videosCollection := vc.DB.Collection("videos")
	filter := bson.M{"_id": videoID, "userId": userID}

	err = videosCollection.FindOne(context.TODO(), filter).Decode(&video)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Video not found or access denied"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve video"})
		log.Printf("Error finding video %s for user %s: %v\n", videoIDHex, userID.Hex(), err)
		return
	}

	c.JSON(http.StatusOK, video)
}

// UpdateVideoMetadata handles updating a video's metadata.
func (vc *VideoController) UpdateVideoMetadata(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	videoIDHex := c.Param("videoId")
	videoID, err := primitive.ObjectIDFromHex(videoIDHex)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid video ID format"})
		return
	}

	var req UpdateVideoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	videosCollection := vc.DB.Collection("videos")
	filter := bson.M{"_id": videoID, "userId": userID}

	// Ensure the video exists and belongs to the user before updating
	var existingVideo models.Video
	err = videosCollection.FindOne(context.TODO(), filter).Decode(&existingVideo)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Video not found or access denied"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify video ownership"})
		return
	}

	update := bson.M{"$set": bson.M{"updatedAt": time.Now()}}
	if req.Title != nil {
		update["$set"].(bson.M)["title"] = *req.Title
	}
	if req.Description != nil {
		update["$set"].(bson.M)["description"] = *req.Description
	}
	if req.Tags != nil { // Allow setting tags to empty array
		update["$set"].(bson.M)["tags"] = req.Tags
	}
	if req.Status != nil {
		// Add validation for allowed status values if necessary
		update["$set"].(bson.M)["status"] = *req.Status
	}

	if len(update["$set"].(bson.M)) == 1 { // Only updatedAt is set
		c.JSON(http.StatusBadRequest, gin.H{"error": "No update fields provided"})
		return
	}


	result, err := videosCollection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update video metadata"})
		log.Printf("Error updating video %s for user %s: %v\n", videoIDHex, userID.Hex(), err)
		return
	}

	if result.MatchedCount == 0 {
		// This case should ideally be caught by the FindOne above, but as a safeguard.
		c.JSON(http.StatusNotFound, gin.H{"error": "Video not found or access denied"})
		return
	}

	// Fetch the updated document to return
	var updatedVideo models.Video
	err = videosCollection.FindOne(context.TODO(), filter).Decode(&updatedVideo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve updated video metadata"})
		return
	}

	c.JSON(http.StatusOK, updatedVideo)
}

// DeleteVideo handles deleting a video and its associated file.
func (vc *VideoController) DeleteVideo(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	videoIDHex := c.Param("videoId")
	videoID, err := primitive.ObjectIDFromHex(videoIDHex)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid video ID format"})
		return
	}

	videosCollection := vc.DB.Collection("videos")
	filter := bson.M{"_id": videoID, "userId": userID}

	// Find the video to get its storage path
	var videoToDelete models.Video
	err = videosCollection.FindOne(context.TODO(), filter).Decode(&videoToDelete)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Video not found or access denied"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve video details for deletion"})
		log.Printf("Error finding video %s for deletion by user %s: %v\n", videoIDHex, userID.Hex(), err)
		return
	}

	// Delete the video file from storage
	if videoToDelete.StoragePath != "" {
		err := os.Remove(videoToDelete.StoragePath)
		if err != nil && !os.IsNotExist(err) { // Log error if it's not "file not found"
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete video file from storage"})
			log.Printf("Error deleting video file %s for video %s: %v\n", videoToDelete.StoragePath, videoIDHex, err)
			return
		}
		if os.IsNotExist(err) {
			log.Printf("Video file %s for video %s not found, proceeding with DB deletion.\n", videoToDelete.StoragePath, videoIDHex)
		}
	} else {
		log.Printf("StoragePath for video %s is empty, skipping file deletion.\n", videoIDHex)
	}


	// Delete the video document from MongoDB
	result, err := videosCollection.DeleteOne(context.TODO(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete video from database"})
		log.Printf("Error deleting video %s from DB for user %s: %v\n", videoIDHex, userID.Hex(), err)
		return
	}

	if result.DeletedCount == 0 {
		// This should ideally be caught by the FindOne above.
		c.JSON(http.StatusNotFound, gin.H{"error": "Video not found in database or access denied"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Video deleted successfully"})
}

// AssignVideoToFolder handles assigning or unassigning a video from a folder.
func (vc *VideoController) AssignVideoToFolder(c *gin.Context) {
	userID, err := getUserIDFromContext(c) // Assumes getUserIDFromContext is available
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	videoIDHex := c.Param("videoId")
	videoID, err := primitive.ObjectIDFromHex(videoIDHex)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid video ID format"})
		return
	}

	var req AssignVideoToFolderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	videosCollection := vc.DB.Collection("videos")
	foldersCollection := vc.DB.Collection("folders") // Needed to verify folder existence

	// Verify video exists and belongs to user
	var video models.Video
	err = videosCollection.FindOne(context.TODO(), bson.M{"_id": videoID, "userId": userID}).Decode(&video)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Video not found or access denied"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve video details"})
		return
	}

	var targetFolderID *primitive.ObjectID // This will be nil if req.FolderID is nil or empty

	if req.FolderID != nil && *req.FolderID != "" {
		if strings.ToLower(*req.FolderID) == "null" || *req.FolderID == "" { // Explicitly unassign if "null" or empty string passed
            targetFolderID = nil
        } else {
            parsedFolderID, err := primitive.ObjectIDFromHex(*req.FolderID)
            if err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid folderId format"})
                return
            }
            // Verify folder exists and belongs to user
            err = foldersCollection.FindOne(context.TODO(), bson.M{"_id": parsedFolderID, "userId": userID}).Err()
            if err != nil {
                if err == mongo.ErrNoDocuments {
                    c.JSON(http.StatusNotFound, gin.H{"error": "Target folder not found or access denied"})
                    return
                }
                c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify target folder"})
                return
            }
            targetFolderID = &parsedFolderID
        }
	} else { // If folderId key is present but value is null, or if key is missing
		targetFolderID = nil 
	}


	// Update video document
	update := bson.M{"$set": bson.M{"updatedAt": time.Now()}}
	if targetFolderID == nil {
		update["$unset"] = bson.M{"folderId": ""} // Remove the folderId field
	} else {
		update["$set"].(bson.M)["folderId"] = targetFolderID
	}
	
	result, err := videosCollection.UpdateOne(context.TODO(), bson.M{"_id": videoID, "userId": userID}, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update video's folder assignment"})
		log.Printf("Error updating video %s folder assignment: %v\n", videoIDHex, err)
		return
	}
	if result.MatchedCount == 0 {
		// Should be caught by initial video check
		c.JSON(http.StatusNotFound, gin.H{"error": "Video not found or access denied (during update)"})
		return
	}

	// Fetch and return updated video document
	var updatedVideo models.Video
	err = videosCollection.FindOne(context.TODO(), bson.M{"_id": videoID, "userId": userID}).Decode(&updatedVideo)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve updated video data"})
        return
    }
	c.JSON(http.StatusOK, updatedVideo)
}
