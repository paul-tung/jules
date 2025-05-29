package controllers

import (
	"auth-service/models"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// FolderController handles folder-related requests.
type FolderController struct {
	DB *mongo.Database
}

// NewFolderController creates a new FolderController.
func NewFolderController(db *mongo.Database) *FolderController {
	return &FolderController{DB: db}
}

// CreateFolderRequest represents the request body for creating a folder.
type CreateFolderRequest struct {
	Name           string `json:"name" binding:"required"`
	ParentFolderID string `json:"parentFolderId"` // Optional, hex string
}

// UpdateFolderRequest represents the request body for updating a folder.
type UpdateFolderRequest struct {
	Name string `json:"name" binding:"required"`
}

// AssignVideoToFolderRequest represents the request body for assigning a video to a folder.
type AssignVideoToFolderRequest struct {
	FolderID *string `json:"folderId"` // Optional hex string, nil means unassign
}


// isFolderNameUnique checks if a folder name is unique for a user at a specific parent level.
func (fc *FolderController) isFolderNameUnique(userID primitive.ObjectID, name string, parentFolderID *primitive.ObjectID) (bool, error) {
	foldersCollection := fc.DB.Collection("folders")
	filter := bson.M{
		"userId":         userID,
		"name":           name,
		"parentFolderId": parentFolderID, // This will be nil for root folders if parentFolderID is nil
	}
	count, err := foldersCollection.CountDocuments(context.TODO(), filter)
	if err != nil {
		return false, err
	}
	return count == 0, nil
}

// CreateFolder handles the creation of a new folder.
func (fc *FolderController) CreateFolder(c *gin.Context) {
	userID, err := getUserIDFromContext(c) // Assuming getUserIDFromContext is available (from video_controller or common utils)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	var req CreateFolderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	var parentID *primitive.ObjectID
	if req.ParentFolderID != "" {
		parsedParentID, err := primitive.ObjectIDFromHex(req.ParentFolderID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid parentFolderId format"})
			return
		}
		// Optional: Check if parent folder exists and belongs to the user
		parentFilter := bson.M{"_id": parsedParentID, "userId": userID}
		count, err := fc.DB.Collection("folders").CountDocuments(context.TODO(), parentFilter)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify parent folder"})
			return
		}
		if count == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Parent folder not found or access denied"})
			return
		}
		parentID = &parsedParentID
	}

	unique, err := fc.isFolderNameUnique(userID, req.Name, parentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check folder name uniqueness"})
		return
	}
	if !unique {
		c.JSON(http.StatusConflict, gin.H{"error": "A folder with this name already exists at this level."})
		return
	}

	folder := models.Folder{
		ID:             primitive.NewObjectID(),
		UserID:         userID,
		Name:           req.Name,
		ParentFolderID: parentID,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	_, err = fc.DB.Collection("folders").InsertOne(context.TODO(), folder)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create folder"})
		return
	}

	c.JSON(http.StatusCreated, folder)
}

// ListFolders handles listing all folders for the authenticated user.
// Supports ?parentFolderId= (hex string or "root" for root folders)
func (fc *FolderController) ListFolders(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	parentFolderIDQuery := c.Query("parentFolderId")
	filter := bson.M{"userId": userID}

	if parentFolderIDQuery != "" {
		if parentFolderIDQuery == "root" {
			filter["parentFolderId"] = nil
		} else {
			parsedParentID, err := primitive.ObjectIDFromHex(parentFolderIDQuery)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid parentFolderId query parameter format"})
				return
			}
			filter["parentFolderId"] = parsedParentID
		}
	}
	// If parentFolderId is not provided, it lists all folders for the user (can be refined).
	// For MVP, if not specified, we can list root folders by default or all folders.
	// Let's default to listing root folders if not specified.
	if _, ok := filter["parentFolderId"]; !ok && parentFolderIDQuery == "" { // Only set to nil if not already set by "root" or specific ID
         filter["parentFolderId"] = nil
    }


	findOptions := options.Find().SetSort(bson.D{{"name", 1}}) // Sort by name
	cursor, err := fc.DB.Collection("folders").Find(context.TODO(), filter, findOptions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve folders"})
		return
	}
	defer cursor.Close(context.TODO())

	var folders []models.Folder
	if err = cursor.All(context.TODO(), &folders); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode folders"})
		return
	}
	if folders == nil {
		folders = []models.Folder{}
	}

	c.JSON(http.StatusOK, folders)
}

// UpdateFolder handles renaming a folder.
func (fc *FolderController) UpdateFolder(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	folderIDHex := c.Param("folderId")
	folderID, err := primitive.ObjectIDFromHex(folderIDHex)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid folder ID format"})
		return
	}

	var req UpdateFolderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	// Find the existing folder to get its ParentFolderID for uniqueness check
	var existingFolder models.Folder
	err = fc.DB.Collection("folders").FindOne(context.TODO(), bson.M{"_id": folderID, "userId": userID}).Decode(&existingFolder)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Folder not found or access denied"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve folder details"})
		return
	}
	
	// If name is not changing, no need to check for uniqueness or update
    if existingFolder.Name == req.Name {
        c.JSON(http.StatusOK, existingFolder) // Return existing folder data
        return
    }

	unique, err := fc.isFolderNameUnique(userID, req.Name, existingFolder.ParentFolderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check folder name uniqueness"})
		return
	}
	if !unique {
		c.JSON(http.StatusConflict, gin.H{"error": "A folder with this name already exists at this level."})
		return
	}

	update := bson.M{
		"$set": bson.M{
			"name":      req.Name,
			"updatedAt": time.Now(),
		},
	}

	result, err := fc.DB.Collection("folders").UpdateOne(context.TODO(), bson.M{"_id": folderID, "userId": userID}, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update folder"})
		return
	}
	if result.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Folder not found or access denied (during update)"})
		return
	}
	
	// Fetch and return updated folder
	var updatedFolder models.Folder
	err = fc.DB.Collection("folders").FindOne(context.TODO(), bson.M{"_id": folderID, "userId": userID}).Decode(&updatedFolder)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve updated folder data"})
        return
    }
	c.JSON(http.StatusOK, updatedFolder)
}

// DeleteFolder handles deleting a folder.
func (fc *FolderController) DeleteFolder(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	folderIDHex := c.Param("folderId")
	folderID, err := primitive.ObjectIDFromHex(folderIDHex)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid folder ID format"})
		return
	}

	// Check if folder exists and belongs to user
	err = fc.DB.Collection("folders").FindOne(context.TODO(), bson.M{"_id": folderID, "userId": userID}).Err()
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Folder not found or access denied"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error checking folder existence"})
		return
	}

	// Check if folder is empty (no videos)
	videoCount, err := fc.DB.Collection("videos").CountDocuments(context.TODO(), bson.M{"folderId": folderID, "userId": userID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check if folder is empty"})
		return
	}
	if videoCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Folder is not empty. It contains %d video(s).", videoCount)})
		return
	}
	
	// Check if folder contains any sub-folders (optional, for stricter deletion)
	// For MVP, we only check videos. If sub-folders should also be checked:
	// subFolderCount, err := fc.DB.Collection("folders").CountDocuments(context.TODO(), bson.M{"parentFolderId": folderID, "userId": userID})
	// if err != nil { ... }
	// if subFolderCount > 0 { c.JSON(http.StatusBadRequest, gin.H{"error": "Folder contains sub-folders."}); return }


	result, err := fc.DB.Collection("folders").DeleteOne(context.TODO(), bson.M{"_id": folderID, "userId": userID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete folder"})
		return
	}
	if result.DeletedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Folder not found or access denied (during delete)"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Folder deleted successfully"})
}

// AssignVideoToFolder handles assigning or unassigning a video from a folder.
// This method will be part of VideoController but is defined here for reference to the request struct.
// Actual implementation will be in video_controller.go
func (vc *VideoController) AssignVideoToFolder(c *gin.Context) { // Note: Receiver is VideoController
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

	var req AssignVideoToFolderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	videosCollection := vc.DB.Collection("videos")
	foldersCollection := vc.DB.Collection("folders")

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

	// Update video document
	update := bson.M{"$set": bson.M{"folderId": targetFolderID, "updatedAt": time.Now()}}
	// If targetFolderID is nil, it effectively unsets or sets to null.
	// If you want to explicitly remove the field if nil:
	// if targetFolderID == nil { update = bson.M{"$unset": bson.M{"folderId": ""}, "$set": bson.M{"updatedAt": time.Now()}} }


	result, err := videosCollection.UpdateOne(context.TODO(), bson.M{"_id": videoID, "userId": userID}, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update video's folder assignment"})
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

// Ensure indexes for folders - typically called from main or db setup
func EnsureFolderIndexes(db *mongo.Database) error {
	foldersCollection := db.Collection("folders")

	// Index for unique folder name per user at the same parent level
	// Note: MongoDB's sparse index in conjunction with compound key handles cases where parentFolderId is null.
	// If parentFolderId is part of the unique key and can be null, a regular unique index will treat nulls as distinct values.
	// To ensure "name" is unique for a user where parentFolderId IS null (root folders), this works.
	// To ensure "name" is unique for a user FOR A SPECIFIC parentFolderId, this also works.
	indexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "userId", Value: 1},
			{Key: "parentFolderId", Value: 1}, // nil will be treated as a specific value for uniqueness
			{Key: "name", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	}
	_, err := foldersCollection.Indexes().CreateOne(context.TODO(), indexModel)
	if err != nil {
		return fmt.Errorf("failed to create unique folder name index: %w", err)
	}
	
	// Index for querying folders by user and parent (optional, if ListFolders is heavily used with parentFolderId)
    // The unique index above might already cover this for reads, but an explicit non-unique index can sometimes be optimized differently by MongoDB.
    // For now, the unique index should be sufficient for performance on typical query patterns.
	// queryIndex := mongo.IndexModel{
	// 	Keys: bson.D{
	// 		{Key: "userId", Value: 1},
	// 		{Key: "parentFolderId", Value: 1},
	// 	},
	// }
	// _, err = foldersCollection.Indexes().CreateOne(context.TODO(), queryIndex)
	// if err != nil {
	// 	return fmt.Errorf("failed to create folder query index: %w", err)
	// }

	return nil
}
