package fiber_inbound_adapter

import (
	"context"

	"github.com/gofiber/fiber/v2"

	inbound_port "go-boilerplate/internal/port/inbound"
)

func InitRoute(
	ctx context.Context,
	app *fiber.App,
	port inbound_port.HttpPort,
) {
	internal := app.Group("/internal")
	internal.Use(func(c *fiber.Ctx) error {
		return port.Middleware().InternalAuth(c)
	})
	internal.Post("/client-upsert", func(c *fiber.Ctx) error {
		return port.Client().Upsert(c)
	})
	internal.Post("/client-find", func(c *fiber.Ctx) error {
		return port.Client().Find(c)
	})
	internal.Delete("/client-delete", func(c *fiber.Ctx) error {
		return port.Client().Delete(c)
	})

	client := app.Group("/v1")
	client.Use(func(c *fiber.Ctx) error {
		return port.Middleware().ClientAuth(c)
	})
	client.Get("/ping", func(c *fiber.Ctx) error {
		return port.Ping().GetResource(c)
	})
}
