package temporal_inbound_adapter

import (
	"context"

	inbound_port "prabogo/internal/port/inbound"
	"prabogo/utils/log"
)

func InitRoute(
	ctx context.Context,
	args []string,
	port inbound_port.WorkflowPort,
) {
	if len(args) > 2 {
		switch args[2] {
		case "upsert_client":
			port.Client().Upsert()
			return
		default:
			log.WithContext(ctx).Info("command not found")
		}
	} else {
		log.WithContext(ctx).Info("command not found")
	}
}
