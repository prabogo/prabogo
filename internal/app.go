package internal

import (
	"context"
	"os"
	"os/signal"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
	joonix "github.com/joonix/log"
	_ "github.com/lib/pq"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sirupsen/logrus"

	command_inbound_adapter "prabogo/internal/adapter/inbound/command"
	fiber_inbound_adapter "prabogo/internal/adapter/inbound/fiber"
	mcp_inbound_adapter "prabogo/internal/adapter/inbound/mcp"
	rabbitmq_inbound_adapter "prabogo/internal/adapter/inbound/rabbitmq"
	temporal_inbound_adapter "prabogo/internal/adapter/inbound/temporal"
	postgres_outbound_adapter "prabogo/internal/adapter/outbound/postgres"
	rabbitmq_outbound_adapter "prabogo/internal/adapter/outbound/rabbitmq"
	redis_outbound_adapter "prabogo/internal/adapter/outbound/redis"
	temporal_outbound_adapter "prabogo/internal/adapter/outbound/temporal"
	"prabogo/internal/domain"
	_ "prabogo/internal/migration/postgres"
	outbound_port "prabogo/internal/port/outbound"
	"prabogo/utils"
	"prabogo/utils/activity"
	"prabogo/utils/database"
	"prabogo/utils/log"
	mcp_utils "prabogo/utils/mcp"
	otel_utils "prabogo/utils/otel"
	"prabogo/utils/rabbitmq"
	"prabogo/utils/redis"
)

var databaseDriverList = []string{"postgres"}
var httpDriverList = []string{"fiber"}
var messageDriverList = []string{"rabbitmq"}
var cacheDriverList = []string{"redis"}
var workflowDriverList = []string{"temporal"}
var mcpTransportList = []string{"http", "stdio"}
var outboundDatabaseDriver string
var outboundMessageDriver string
var outboundCacheDriver string
var outboundWorkflowDriver string
var inboundHttpDriver string
var inboundMessageDriver string
var inboundWorkflowDriver string
var inboundMcpTransport string

type App struct {
	ctx      context.Context
	domain   domain.Domain
	otelShut func(context.Context) error
}

func NewApp() *App {
	ctx := activity.NewContext("init")
	ctx = activity.WithClientID(ctx, "system")
	_ = godotenv.Load(".env")

	otelShutdown, err := otel_utils.Setup(ctx)
	if err != nil {
		log.WithContext(ctx).Warnf("failed to initialize opentelemetry: %v", err)
	}
	configureLogging()

	outboundDatabaseDriver = os.Getenv("OUTBOUND_DATABASE_DRIVER")
	outboundMessageDriver = os.Getenv("OUTBOUND_MESSAGE_DRIVER")
	outboundCacheDriver = os.Getenv("OUTBOUND_CACHE_DRIVER")
	outboundWorkflowDriver = os.Getenv("OUTBOUND_WORKFLOW_DRIVER")
	inboundHttpDriver = os.Getenv("INBOUND_HTTP_DRIVER")
	inboundMessageDriver = os.Getenv("INBOUND_MESSAGE_DRIVER")
	inboundWorkflowDriver = os.Getenv("INBOUND_WORKFLOW_DRIVER")
	inboundMcpTransport = os.Getenv("INBOUND_MCP_TRANSPORT")

	domain := domain.NewDomain(
		databaseOutbound(ctx),
		messageOutbound(ctx),
		cacheOutbound(ctx),
		workflowOutbound(ctx),
	)

	return &App{
		ctx:      ctx,
		domain:   domain,
		otelShut: otelShutdown,
	}
}

func (a *App) Run(option string) {
	switch option {
	case "http":
		a.httpInbound()
	case "message":
		a.messageInbound()
	case "workflow":
		a.workflowInbound()
	case "mcp":
		a.mcpInbound()
	default:
		a.commandInbound()
	}
}

func databaseOutbound(ctx context.Context) outbound_port.DatabasePort {
	if !utils.IsInList(databaseDriverList, outboundDatabaseDriver) {
		if outboundDatabaseDriver == "" {
			return nil
		}

		log.WithContext(ctx).Fatal("database driver is not supported")
		os.Exit(1)
	}

	isUseMigration := false
	db := database.InitDatabase(ctx, outboundDatabaseDriver, isUseMigration)

	switch outboundDatabaseDriver {
	case "postgres":
		return postgres_outbound_adapter.NewAdapter(db)
	}
	return nil
}

func messageOutbound(ctx context.Context) outbound_port.MessagePort {
	if !utils.IsInList(messageDriverList, outboundMessageDriver) {
		if outboundMessageDriver == "" {
			return nil
		}

		log.WithContext(ctx).Fatal("message driver is not supported")
		os.Exit(1)
	}

	switch outboundMessageDriver {
	case "rabbitmq":
		if err := rabbitmq.InitMessage(); err != nil {
			log.WithContext(ctx).Fatalf("failed to init rabbitmq: %v", err)
		}
		return rabbitmq_outbound_adapter.NewAdapter()
	}
	return nil
}

func cacheOutbound(ctx context.Context) outbound_port.CachePort {
	if !utils.IsInList(cacheDriverList, outboundCacheDriver) {
		if outboundCacheDriver == "" {
			return nil
		}

		log.WithContext(ctx).Fatal("cache driver is not supported")
		os.Exit(1)
	}

	switch outboundCacheDriver {
	case "redis":
		redis.InitDatabase()
		return redis_outbound_adapter.NewAdapter()
	}
	return nil
}

func workflowOutbound(ctx context.Context) outbound_port.WorkflowPort {
	if !utils.IsInList(workflowDriverList, outboundWorkflowDriver) {
		if outboundWorkflowDriver == "" {
			return nil
		}

		log.WithContext(ctx).Fatal("workflow driver is not supported")
		os.Exit(1)
	}

	switch outboundWorkflowDriver {
	case "temporal":
		return temporal_outbound_adapter.NewAdapter()
	}
	return nil
}

func (a *App) httpInbound() {
	ctx := a.ctx
	if !utils.IsInList(httpDriverList, inboundHttpDriver) {
		log.WithContext(ctx).Fatal("http driver is not supported")
		os.Exit(1)
	}

	switch inboundHttpDriver {
	case "fiber":
		app := fiber.New()
		inboundHttpAdapter := fiber_inbound_adapter.NewAdapter(a.domain)
		fiber_inbound_adapter.InitRoute(ctx, app, inboundHttpAdapter)

		shutdownSignal := make(chan os.Signal, 1)
		signal.Notify(shutdownSignal, os.Interrupt)
		defer signal.Stop(shutdownSignal)

		go func() {
			<-shutdownSignal
			log.WithContext(ctx).Info("gracefully shutting down http server")

			if err := app.Shutdown(); err != nil {
				log.WithContext(ctx).Errorf("failed to shutdown http server: %+v", err)
			}

			if a.otelShut != nil {
				if err := a.otelShut(ctx); err != nil {
					log.WithContext(ctx).Errorf("failed to shutdown opentelemetry: %+v", err)
				}
			}
		}()

		if err := app.Listen(":" + os.Getenv("SERVER_PORT")); err != nil {
			log.WithContext(ctx).Fatalf("failed to listen and serve: %+v", err)
		}

		log.WithContext(ctx).Info("http server stopped")
	}
}

func (a *App) messageInbound() {
	ctx := a.ctx
	if !utils.IsInList(messageDriverList, inboundMessageDriver) {
		log.WithContext(ctx).Fatal("message driver is not supported")
		os.Exit(1)
	}

	switch inboundMessageDriver {
	case "rabbitmq":
		inboundMessageAdapter := rabbitmq_inbound_adapter.NewAdapter(a.domain)
		rabbitmq_inbound_adapter.InitRoute(ctx, os.Args, inboundMessageAdapter)
	}
}

func (a *App) commandInbound() {
	ctx := a.ctx
	inboundCommandAdapter := command_inbound_adapter.NewAdapter(a.domain)
	command_inbound_adapter.InitRoute(ctx, os.Args, inboundCommandAdapter)

	if a.otelShut != nil {
		if err := a.otelShut(ctx); err != nil {
			log.WithContext(ctx).Errorf("failed to shutdown opentelemetry: %+v", err)
		}
	}
}

func (a *App) workflowInbound() {
	ctx := a.ctx
	if !utils.IsInList(workflowDriverList, inboundWorkflowDriver) {
		log.WithContext(ctx).Fatal("workflow driver is not supported")
		os.Exit(1)
	}

	switch inboundWorkflowDriver {
	case "temporal":
		inboundWorkflowAdapter := temporal_inbound_adapter.NewAdapter(a.domain)
		temporal_inbound_adapter.InitRoute(ctx, os.Args, inboundWorkflowAdapter)
	}
}

func (a *App) mcpInbound() {
	ctx := a.ctx
	if !utils.IsInList(mcpTransportList, inboundMcpTransport) {
		log.WithContext(ctx).Fatal("mcp transport is not supported")
		os.Exit(1)
	}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "prabogo-mcp",
		Version: "1.0.0",
	}, nil)

	inboundMcpAdapter := mcp_inbound_adapter.NewAdapter(a.domain)
	mcp_inbound_adapter.InitRoute(ctx, server, inboundMcpAdapter)

	switch inboundMcpTransport {
	case "http":
		mcp_utils.RunHTTPServer(ctx, server)
	case "stdio":
		mcp_utils.RunStdioServer(ctx, server)
	}
}

func configureLogging() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.AddHook(utils.LogrusSourceContextHook{})
	// MaskingHook must be registered before any log-emitting hook so it can
	// rewrite entry.Data in place before the stdout formatter runs and before
	// the OTLP hook exports the record.
	logrus.AddHook(log.NewMaskingHook())

	if os.Getenv("OTEL_LOG_ENABLED") == "true" {
		logrus.AddHook(log.NewOTelLogrusHook())
	}

	logrus.SetFormatter(&joonix.FluentdFormatter{})
}
