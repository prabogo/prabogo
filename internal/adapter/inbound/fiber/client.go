package fiber_inbound_adapter

import (
	"context"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/palantir/stacktrace"

	"prabogo/internal/domain"
	"prabogo/internal/model"
	inbound_port "prabogo/internal/port/inbound"
	"prabogo/utils/activity"
	"prabogo/utils/log"
)

type clientAdapter struct {
	domain domain.Domain
}

const (
	activityNameUpsert         = "http_client_upsert"
	activityNameFindByFilter   = "http_client_find_by_filter"
	activityNameDeleteByFilter = "http_client_delete_by_filter"
)

func NewClientAdapter(
	domain domain.Domain,
) inbound_port.ClientHttpPort {
	return &clientAdapter{
		domain: domain,
	}
}

func (h *clientAdapter) Upsert(a any) error {
	c := a.(*fiber.Ctx)
	ctx := newRequestContext(c, activityNameUpsert)
	var payload []model.ClientInput
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.Response{
			Success: false,
			Error:   err.Error(),
		})
	}
	ctx = context.WithValue(ctx, activity.Payload, payload)

	ctx, span := startDomainSpan(ctx, activityNameUpsert)
	defer span.End()

	results, err := h.domain.Client().Upsert(ctx, payload)
	if err != nil {
		addSpanAttribute(c, "error.message", stacktrace.RootCause(err).Error())
		log.WithContext(ctx).Errorf("http client upsert failed: %s", stacktrace.RootCause(err).Error())
		return c.Status(fiber.StatusInternalServerError).JSON(model.Response{
			Success: false,
			Error:   fmt.Sprintf("http client upsert failed: %s", stacktrace.RootCause(err).Error()),
		})
	}

	log.WithContext(ctx).Info("http client upsert success")
	return c.JSON(model.Response{
		Success: true,
		Data:    results,
	})
}

func (h *clientAdapter) Find(a any) error {
	c := a.(*fiber.Ctx)
	ctx := newRequestContext(c, activityNameFindByFilter)
	var payload model.ClientFilter
	if err := c.BodyParser(&payload); err != nil {
		log.WithContext(ctx).Errorf("http client find by filter failed: %s", stacktrace.RootCause(err).Error())
		return c.Status(fiber.StatusBadRequest).JSON(model.Response{
			Success: false,
			Error:   err.Error(),
		})
	}
	ctx = context.WithValue(ctx, activity.Payload, payload)

	ctx, span := startDomainSpan(ctx, activityNameFindByFilter)
	defer span.End()

	results, err := h.domain.Client().FindByFilter(ctx, payload)
	if err != nil {
		addSpanAttribute(c, "error.message", stacktrace.RootCause(err).Error())
		log.WithContext(ctx).Errorf("http client find by filter failed: %s", stacktrace.RootCause(err).Error())
		return c.Status(fiber.StatusInternalServerError).JSON(model.Response{
			Success: false,
			Error:   fmt.Sprintf("http client find by filter failed: %s", stacktrace.RootCause(err).Error()),
		})
	}

	ctx = context.WithValue(ctx, activity.Result, results)
	log.WithContext(ctx).Info("http client find success")
	return c.JSON(model.Response{
		Success: true,
		Data:    results,
	})
}

func (h *clientAdapter) Delete(a any) error {
	c := a.(*fiber.Ctx)
	ctx := newRequestContext(c, activityNameDeleteByFilter)
	var payload model.ClientFilter
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.Response{
			Success: false,
			Error:   err.Error(),
		})
	}
	ctx = context.WithValue(ctx, activity.Payload, payload)

	ctx, span := startDomainSpan(ctx, activityNameDeleteByFilter)
	defer span.End()

	err := h.domain.Client().DeleteByFilter(ctx, payload)
	if err != nil {
		addSpanAttribute(c, "error.message", stacktrace.RootCause(err).Error())
		return c.Status(fiber.StatusInternalServerError).JSON(model.Response{
			Success: false,
			Error:   fmt.Sprintf("http client delete by filter failed: %s", stacktrace.RootCause(err).Error()),
		})
	}

	return c.JSON(model.Response{
		Success: true,
	})
}
