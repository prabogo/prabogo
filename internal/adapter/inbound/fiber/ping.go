package fiber_inbound_adapter

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"

	inbound_port "go-boilerplate/internal/port/inbound"
	"go-boilerplate/utils"
)

type pingAdapter struct{}

func NewPingAdapter() inbound_port.PingHttpPort {
	return &pingAdapter{}
}

func (h *pingAdapter) GetResource(a any) error {
	c := a.(*fiber.Ctx)
	idle0, total0 := utils.GetCPUSample()
	time.Sleep(1 * time.Second)
	idle1, total1 := utils.GetCPUSample()

	idleTicks := float64(idle1 - idle0)
	totalTicks := float64(total1 - total0)
	cpuUsage := 100 * (totalTicks - idleTicks) / totalTicks

	total, free, buffers, cached := utils.GetMemorySample()
	coreCount := utils.GetCoreSample()

	return c.JSON(fiber.Map{
		"message": "pong",
		"core": []fiber.Map{
			{"core": fmt.Sprintf("%d Core", coreCount)},
		},
		"cpu": []fiber.Map{
			{
				"usage": fmt.Sprintf("%f %%", cpuUsage),
				"busy":  fmt.Sprintf("%f %%", totalTicks-idleTicks),
				"total": fmt.Sprintf("%f %%", totalTicks),
			},
		},
		"memory": []fiber.Map{
			{
				"usage":  fmt.Sprintf("%f %%", 100*(1-float64(free)/float64(total))),
				"total":  fmt.Sprintf("%f MB", float64(total)/1024),
				"free":   fmt.Sprintf("%f MB", float64(free)/1024),
				"buffer": fmt.Sprintf("%f MB", float64(buffers)/1024),
				"cached": fmt.Sprintf("%f MB", float64(cached)/1024),
			},
		},
	})
}
