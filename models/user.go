package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

// User represents a user in the database.
type User struct {
	ID           primitive.ObjectID `bson:"_id,omitempty"`
	Email        string             `bson:"email,omitempty"`
	PasswordHash string             `bson:"passwordHash,omitempty"`
	Username     string             `bson:"username,omitempty"`
	CreatedAt    time.Time          `bson:"createdAt,omitempty"`
	UpdatedAt    time.Time          `bson:"updatedAt,omitempty"`
}
