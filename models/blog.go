package models

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Blog struct {
	Id          uuid.UUID  `db:"id"`
	Date        time.Time  `db:"date"`
	Title       string     `db:"title"`
	Content     string     `db:"content"`
	PublishedAt *time.Time `db:"published_at"`
	CreatedAt   time.Time  `db:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"`
}

func (b *Blog) IsPublished() bool {
	return b.PublishedAt != nil
}

func (b *Blog) IsBeforeDate() bool {
	return b.Date.After(time.Now())
}

var previewLength = 200

func (b *Blog) HasPreview() bool {
	preview := []rune(b.Content)
	return len(preview) > previewLength
}

func (b *Blog) ContentPreview() string {
	if !b.HasPreview() {
		return b.Content
	}
	preview := []rune(b.Content)
	return fmt.Sprint(string(preview[:previewLength]), "...")
}

func (b *Blog) ToMap() fiber.Map {
	return fiber.Map{
		"Id":             b.Id,
		"Date":           b.Date,
		"Title":          b.Title,
		"Content":        b.Content,
		"PublishedAt":    b.PublishedAt,
		"CreatedAt":      b.CreatedAt,
		"UpdatedAt":      b.UpdatedAt,
		"IsPublished":    b.IsPublished(),
		"IsBeforeDate":   b.IsBeforeDate(),
		"HasPreview":     b.HasPreview(),
		"ContentPreview": b.ContentPreview(),
	}
}
