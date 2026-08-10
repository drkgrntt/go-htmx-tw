package controllers

import (
	"net/http"
	"time"

	"github.com/drkgrntt/htmx-test/database"
	"github.com/drkgrntt/htmx-test/handlers"
	"github.com/drkgrntt/htmx-test/models"
	"github.com/drkgrntt/htmx-test/utils"
	"github.com/gofiber/fiber/v2"
)

type BlogController struct {
}

func NewBlogController(views fiber.Router, api fiber.Router) *BlogController {
	ac := &BlogController{}
	ac.registerViewRoutes(views)
	ac.registerApiRoutes(api)

	return ac
}

func (c *BlogController) registerViewRoutes(views fiber.Router) {
	blog := views.Group("/blog")
	blog.Get("/", handlers.BlogPage)
	blog.Get("/:id", handlers.BlogPostPage)
	blog.Get("/:id/edit", handlers.BlogPostEditPage)
}

func (c *BlogController) registerApiRoutes(api fiber.Router) {
	blog := api.Group("/blog")

	blog.Post("/", c.createBlog)
	blog.Put("/:id", c.updateBlog)
}

type blogPost struct {
	Date        string `form:"date"`
	Title       string `form:"title"`
	Content     string `form:"content"`
	Publish     bool   `form:"publish"`
	PublishDate *time.Time
}

func (c *BlogController) createBlog(ctx *fiber.Ctx) error {
	config := utils.GetConfig()
	if config.Environment != "development" {
		ctx.Status(http.StatusUnauthorized)
		return nil
	}

	body := blogPost{}
	err := ctx.BodyParser(&body)
	utils.LogFatalError(err)

	if body.Date == "" {
		ctx.Status(http.StatusBadRequest)
		return nil
	}

	if body.Publish {
		now := time.Now()
		body.PublishDate = &now
	}

	db := database.GetDatabase()
	db.MustExec(
		"INSERT INTO blogs (date, title, content, published_at) VALUES ($1, $2, $3, $4);",
		&body.Date,
		&body.Title,
		&body.Content,
		&body.PublishDate,
	)

	var newBlog models.Blog
	err = db.Get(&newBlog, "SELECT * FROM blogs ORDER BY created_at DESC LIMIT 1")
	utils.LogFatalError(err)

	return handlers.BlogPost(ctx, &newBlog)
}

func (c *BlogController) updateBlog(ctx *fiber.Ctx) error {
	config := utils.GetConfig()
	if config.Environment != "development" {
		ctx.Status(http.StatusUnauthorized)
		return nil
	}

	db := database.GetDatabase()
	var blog models.Blog
	err := db.Get(&blog, "SELECT * FROM blogs WHERE id = $1 LIMIT 1", ctx.Params("id"))
	utils.LogFatalError(err)

	body := blogPost{}
	err = ctx.BodyParser(&body)
	utils.LogFatalError(err)

	if body.Date == "" {
		ctx.Status(http.StatusBadRequest)
		return nil
	}

	body.PublishDate = blog.PublishedAt

	if body.Publish && !blog.IsPublished() {
		now := time.Now()
		body.PublishDate = &now
	}

	db.MustExec(
		"UPDATE blogs SET date = $1, title = $2, content = $3, published_at = $4, updated_at = $5 WHERE id = $6;",
		&body.Date,
		&body.Title,
		&body.Content,
		&body.PublishDate,
		time.Now(),
		ctx.Params("id"),
	)

	err = db.Get(&blog, "SELECT * FROM blogs WHERE id = $1 LIMIT 1", ctx.Params("id"))
	utils.LogFatalError(err)

	return ctx.Status(http.StatusOK).JSON(blog)
}
