package fiber_inbound_adapter

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/palantir/stacktrace"

	"go-boilerplate/internal/domain"
	"go-boilerplate/internal/model"
	inbound_port "go-boilerplate/internal/port/inbound"
	"go-boilerplate/utils/activity"
)

type clientAdapter struct {
	domain domain.Domain
}

func NewClientAdapter(
	domain domain.Domain,
) inbound_port.ClientHttpPort {
	return &clientAdapter{
		domain: domain,
	}
}

func (h *clientAdapter) Upsert(a any) error {
	c := a.(*fiber.Ctx)
	ctx := activity.NewContext("http_client_upsert")
	var payload []model.ClientInput
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.Response{
			Success: false,
			Error:   err.Error(),
		})
	}
	ctx = context.WithValue(ctx, activity.Payload, payload)

	results, err := h.domain.Client().Upsert(ctx, payload)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.Response{
			Success: false,
			Error:   stacktrace.RootCause(err).Error(),
		})
	}

	return c.JSON(model.Response{
		Success: true,
		Data:    results,
	})
}

func (h *clientAdapter) Find(a any) error {
	c := a.(*fiber.Ctx)
	ctx := activity.NewContext("http_client_find_by_filter")
	var payload model.ClientFilter
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.Response{
			Success: false,
			Error:   err.Error(),
		})
	}
	ctx = context.WithValue(ctx, activity.Payload, payload)

	results, err := h.domain.Client().FindByFilter(ctx, payload)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.Response{
			Success: false,
			Error:   stacktrace.RootCause(err).Error(),
		})
	}

	return c.JSON(model.Response{
		Success: true,
		Data:    results,
	})
}

func (h *clientAdapter) Delete(a any) error {
	c := a.(*fiber.Ctx)
	ctx := activity.NewContext("http_client_delete_by_filter")
	var payload model.Request
	payload.Data = model.ClientFilter{}
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.Response{
			Success: false,
			Error:   err.Error(),
		})
	}
	ctx = context.WithValue(ctx, activity.Payload, payload)

	err := h.domain.Client().DeleteByFilter(ctx, payload.Data.(model.ClientFilter))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.Response{
			Success: false,
			Error:   stacktrace.RootCause(err).Error(),
		})
	}

	return c.JSON(model.Response{
		Success: true,
	})
}
