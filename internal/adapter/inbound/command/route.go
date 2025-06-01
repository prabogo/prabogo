package command_inbound_adapter

import (
	"context"

	inbound_port "prabogo/internal/port/inbound"
	"prabogo/utils/log"
)

func InitRoute(
	ctx context.Context,
	args []string,
	port inbound_port.CommandPort,
) {
	if len(args) > 2 {
		switch args[1] {
		case "publish_upsert_client":
			// name := args[2]
			// port.Client().PublishUpsert(name)
		default:
			log.WithContext(ctx).Info("command not found")
		}
	} else {
		log.WithContext(ctx).Info("command not found")
	}
}
