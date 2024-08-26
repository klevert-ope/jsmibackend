package models

import (
	"github.com/google/uuid"
	"time"
)

type Post struct {
	ID        uuid.UUID `json:"id"`
	Title     string    `json:"title"`
	Excerpt   string    `json:"excerpt"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}
