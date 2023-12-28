package main

import (
	"fmt"
	"log"
	"time"

	"github.com/drkgrntt/htmx-test/controllers"
	"github.com/drkgrntt/htmx-test/database"
	"github.com/drkgrntt/htmx-test/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
)

func main() {
	config, err := utils.LoadConfig(".")
	if err != nil {
		log.Fatal("? Could not load environment variables", err)
	}

	database.Connect()

	app := fiber.New(fiber.Config{
		Views:             html.New("./views", ".html"),
		ViewsLayout:       "layout/main",
		PassLocalsToViews: true,
	})

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("Year", time.Now().Year())
		c.Locals("Environment", config.Environment)
		return c.Next()
	})

	controllers.InitializeControllers(app)

	log.Fatal(app.Listen(fmt.Sprintf(":%s", config.ServerPort)))
}
