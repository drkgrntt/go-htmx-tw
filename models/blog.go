package models

import (
	"time"

	"github.com/google/uuid"
)

type Blog struct {
	Id      uuid.UUID `db:"id"`
	Date    time.Time `db:"date"`
	Title   string    `db:"title"`
	Content string    `db:"content"`
}
