package controllers

import (
	"net/http"
	"time"

	"github.com/drkgrntt/htmx-test/database"
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

	blog.Get("/", c.blogsPage)
	blog.Get("/:id", c.blogPage)
}

func (c *BlogController) blogsPage(ctx *fiber.Ctx) error {
	db := database.GetDatabase()
	config := utils.GetConfig()

	blogs := []models.Blog{}
	query := `
		SELECT * FROM blogs
		WHERE published_at IS NOT NULL
		AND date < NOW()
		ORDER BY date DESC;
	`

	if config.Environment == "development" {
		query = `
			SELECT * FROM blogs
			ORDER BY date DESC;
		`
	}

	err := db.Select(&blogs, query)
	utils.LogFatalError(err)

	return ctx.Render("blog", fiber.Map{"Blogs": blogs})
}

func (c *BlogController) blogPage(ctx *fiber.Ctx) error {
	db := database.GetDatabase()
	config := utils.GetConfig()

	blog := models.Blog{}
	query := `
		SELECT * FROM blogs
		WHERE id = $1
		AND published_at IS NOT NULL
		AND date < NOW();
	`

	if config.Environment == "development" {
		query = `
			SELECT * FROM blogs
			WHERE id = $1;
		`
	}

	err := db.Get(&blog, query, ctx.Params("id"))
	if err != nil {
		ctx.Status(http.StatusNotFound)
		return nil
	}

	return ctx.Render("blog-show", fiber.Map{"Blog": blog})
}

func (c *BlogController) registerApiRoutes(api fiber.Router) {
	blog := api.Group("/blog")

	blog.Post("/", c.createBlog)
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

	return ctx.Render("partials/blog-post", newBlog.ToMap(), "")
}
