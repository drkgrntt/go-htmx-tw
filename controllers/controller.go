package controllers

import (
	"github.com/drkgrntt/htmx-test/handlers"
	"github.com/drkgrntt/htmx-test/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/monitor"
)

type Initializer interface {
	RegisterRoutes(app *fiber.App)
}

type Controller struct {
}

func createController() Controller {
	return Controller{}
}

func InitializeControllers(app *fiber.App) {
	mainController := createController()
	mainController.registerRoutes(app)
}

func (c *Controller) registerRoutes(app *fiber.App) {
	views := app.Group("/")
	views.Get("metrics", monitor.New())

	views.Get("/", handlers.LandingPage)

	api := app.Group("/api")
	api.Use(logger.New())

	NewContactController(views, api)
	NewBlogController(views, api)

	// Route to display all routes.
	config := utils.GetConfig()
	// TODO: This could be massaged to be a decent API documentation
	if config.Environment != "production" {
		views.Get("/routes", func(c *fiber.Ctx) error {
			routes := app.GetRoutes(true)
			routeMap := make(map[string][]string)
			for _, route := range routes {
				routeMap[route.Path] = append(routeMap[route.Path], route.Method)
			}
			return c.JSON(fiber.Map{"routes": routeMap})
		})
	}
}
