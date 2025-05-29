package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

// Folder represents a folder in the database.
type Folder struct {
	ID             primitive.ObjectID  `bson:"_id,omitempty"`
	UserID         primitive.ObjectID  `bson:"userId,omitempty"` // Link to the User who created the folder
	Name           string              `bson:"name,omitempty"`
	ParentFolderID *primitive.ObjectID `bson:"parentFolderId,omitempty"` // Pointer to allow null for root folders
	CreatedAt      time.Time           `bson:"createdAt,omitempty"`
	UpdatedAt      time.Time           `bson:"updatedAt,omitempty"`
}
