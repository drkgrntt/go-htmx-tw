package controllers

import (
	"log"
	"net/http"

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
}

func (c *BlogController) blogsPage(ctx *fiber.Ctx) error {
	db := database.GetDatabase()

	blogs := []models.Blog{}
	db.Select(&blogs, "SELECT * FROM blogs ORDER BY date DESC")

	return ctx.Render("blog", fiber.Map{"Blogs": blogs})
}

func (c *BlogController) registerApiRoutes(api fiber.Router) {
	blog := api.Group("/blog")

	blog.Post("/", c.createBlog)
}

type BlogPost struct {
	Date    string `form:"date"`
	Title   string `form:"title"`
	Content string `form:"content"`
}

func (c *BlogController) createBlog(ctx *fiber.Ctx) error {
	config := utils.GetConfig()
	if config.Environment != "development" {
		ctx.Status(http.StatusUnauthorized)
		return nil
	}

	body := new(BlogPost)
	err := ctx.BodyParser(body)
	if err != nil {
		log.Fatal(err)
	}

	db := database.GetDatabase()
	res, err := db.Exec("INSERT INTO blogs (date, title, content) VALUES ($1, $2, $3);", body.Date, body.Title, body.Content)
	if err != nil {
		log.Fatal(err)
	}
	log.Println(res)

	return ctx.Status(http.StatusNoContent).JSON("Dunzo")
}
