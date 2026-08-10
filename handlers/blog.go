package handlers

import (
	"github.com/a-h/templ"
	"github.com/drkgrntt/htmx-test/components/blog"
	"github.com/drkgrntt/htmx-test/models"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
)

func BlogPage(c *fiber.Ctx) error {
	return adaptor.HTTPHandler(templ.Handler(blog.Page(c)))(c)
}

func BlogPostPage(c *fiber.Ctx) error {
	return adaptor.HTTPHandler(templ.Handler(blog.Post(c)))(c)
}

func BlogPost(c *fiber.Ctx, blogPost *models.Blog) error {
	return adaptor.HTTPHandler(templ.Handler(blog.BlogPost(c, blogPost)))(c)
}

func BlogPostEditPage(c *fiber.Ctx) error {
	return adaptor.HTTPHandler(templ.Handler(blog.Edit(c)))(c)
}
