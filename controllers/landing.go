package controllers

import (
	"github.com/drkgrntt/htmx-test/handlers"
	"github.com/gofiber/fiber/v2"
)

type LandingController struct {
}

func NewLandingController(views fiber.Router, api fiber.Router) *LandingController {
	lc := &LandingController{}
	lc.registerViewRoutes(views)
	lc.registerApiRoutes(api)

	return lc
}

func (c *LandingController) registerViewRoutes(views fiber.Router) {
	views.Get("/", handlers.LandingPage)
	views.Get("/tech", handlers.TechPage)
	views.Get("/music", handlers.MusicPage)
}

func (c *LandingController) registerApiRoutes(api fiber.Router) {
}
