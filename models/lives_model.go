package models

import (
	"time"

	"github.com/google/uuid"
)

type Live struct {
	ID        uuid.UUID `json:"id"`
	Title     string    `json:"title"`
	Link      string    `json:"link"`
	CreatedAt time.Time `json:"created_at"`
}
