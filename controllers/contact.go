package controllers

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/drkgrntt/htmx-test/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/mailgun/mailgun-go/v4"
)

type ContactController struct {
}

func NewContactController(views fiber.Router, api fiber.Router) *ContactController {
	ac := &ContactController{}
	ac.registerViewRoutes(views)
	ac.registerApiRoutes(api)

	return ac
}

func (c *ContactController) registerViewRoutes(views fiber.Router) {
	contact := views.Group("/contact")

	contact.Get("/", func(c *fiber.Ctx) error {
		return c.Render("contact", fiber.Map{})
	})

}

func (c *ContactController) registerApiRoutes(api fiber.Router) {
	contact := api.Group("/contact")

	type ContactInfo struct {
		Name    string `form:"name"`
		Email   string `form:"email"`
		Phone   string `form:"phone"`
		Message string `form:"message"`
	}
	contact.Post("/", func(c *fiber.Ctx) error {
		body := new(ContactInfo)
		err := c.BodyParser(body)
		fmt.Println(body, err)

		config := utils.GetConfig()

		mg := mailgun.NewMailgun(config.MgDomain, config.MgApiKey)
		sender := "Contact Form <contact@derekgarnett.com>"
		subject := "New Contact!"
		emailBody := fmt.Sprintf(
			"New Message from %s\n%s\n%s\n\n%s",
			body.Name,
			body.Email,
			body.Phone,
			body.Message,
		)
		recipient := config.RecipientEmail

		// The message object allows you to add attachments and Bcc recipients
		message := mg.NewMessage(sender, subject, emailBody, recipient)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		// Send the message with a 10 second timeout
		resp, id, err := mg.Send(ctx, message)

		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("ID: %s Resp: %s\n", id, resp)

		return c.SendString("Thank you for reaching out! I will get back to you soon.")
	})
}
