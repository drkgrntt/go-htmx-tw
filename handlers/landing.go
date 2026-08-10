package handlers

import (
	"github.com/a-h/templ"
	"github.com/drkgrntt/htmx-test/components/landing"
	"github.com/drkgrntt/htmx-test/components/music"
	"github.com/drkgrntt/htmx-test/components/tech"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
)

func LandingPage(c *fiber.Ctx) error {
	return adaptor.HTTPHandler(templ.Handler(landing.Page(c)))(c)
}

func TechPage(c *fiber.Ctx) error {
	return adaptor.HTTPHandler(templ.Handler(tech.Page(c)))(c)
}

func MusicPage(c *fiber.Ctx) error {
	return adaptor.HTTPHandler(templ.Handler(music.Page(c)))(c)
}
