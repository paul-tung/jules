package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

// Video represents a video in the database.
type Video struct {
	ID               primitive.ObjectID `bson:"_id,omitempty"`
	UserID           primitive.ObjectID `bson:"userId,omitempty"` // Link to the User who uploaded the video
	Title            string             `bson:"title,omitempty"`
	OriginalFilename string             `bson:"originalFilename,omitempty"`
	StoragePath      string             `bson:"storagePath,omitempty"` // Path where the video is stored
	Status           string             `bson:"status,omitempty"`    // e.g., "uploaded", "processing", "active"
	FolderID         *primitive.ObjectID `bson:"folderId,omitempty"`    // Link to the Folder, pointer to allow null
	UploadedAt       time.Time          `bson:"uploadedAt,omitempty"`
	UpdatedAt        time.Time          `bson:"updatedAt,omitempty"`
	Description      string             `bson:"description,omitempty"` // Optional
	Tags             []string           `bson:"tags,omitempty"`        // Optional
	DurationSeconds  float64            `bson:"durationSeconds,omitempty"` // Optional
	ThumbnailPath    string             `bson:"thumbnailPath,omitempty"` // Optional
	Views            int64              `bson:"views,omitempty"`       // Optional
}
