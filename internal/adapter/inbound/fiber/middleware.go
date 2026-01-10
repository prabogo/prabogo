package fiber_inbound_adapter

import (
	"os"

	"github.com/gofiber/fiber/v2"

	"prabogo/internal/domain"
	"prabogo/internal/model"
	"prabogo/utils/activity"
)

const (
	authorizationHeader = "Authorization"
	bearerPrefix        = "Bearer "
	bearerPrefixLen     = 7
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
	authHeader := c.Get(authorizationHeader)
	var bearerToken string
	if len(authHeader) > bearerPrefixLen && authHeader[:bearerPrefixLen] == bearerPrefix {
		bearerToken = authHeader[bearerPrefixLen:]
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
	authHeader := c.Get(authorizationHeader)
	var bearerToken string
	if len(authHeader) > bearerPrefixLen && authHeader[:bearerPrefixLen] == bearerPrefix {
		bearerToken = authHeader[bearerPrefixLen:]
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
