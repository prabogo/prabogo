package fiber_inbound_adapter

import (
	"os"
	"prabogo/internal/domain"
	"prabogo/internal/model"
	"prabogo/utils/activity"

	"github.com/gofiber/fiber/v2"
)

type MiddlewareAdapter interface {
	InternalAuth(a any) error
	ClientAuth(a any) error
}

type middlewareAdapter struct {
	domain domain.Domain
}

func NewMiddlewareAdapter(
	domain domain.Domain,
) MiddlewareAdapter {
	return &middlewareAdapter{
		domain: domain,
	}
}

func (h *middlewareAdapter) InternalAuth(a any) error {
	c := a.(*fiber.Ctx)
	authHeader := c.Get("Authorization")
	var bearerToken string
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		bearerToken = authHeader[7:]
	}

	if bearerToken == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	if bearerToken != os.Getenv("INTERNAL_KEY") {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	return c.Next()
}

func (h *middlewareAdapter) ClientAuth(a any) error {
	c := a.(*fiber.Ctx)
	ctx := activity.NewContext("http_client_auth")
	authHeader := c.Get("Authorization")
	var bearerToken string
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		bearerToken = authHeader[7:]
	}

	if bearerToken == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(model.Response{
			Success: false,
			Error:   "Unauthorized",
		})
	}

	exists, err := h.domain.Client().IsExists(ctx, bearerToken)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.Response{
			Success: false,
			Error:   err.Error(),
		})
	}

	if !exists {
		return c.Status(fiber.StatusUnauthorized).JSON(model.Response{
			Success: false,
			Error:   "Unauthorized",
		})
	}

	return c.Next()
}
