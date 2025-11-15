package di

import (
	"consentcc/config"
	controller "consentcc/internal/controller/http"
	"consentcc/internal/controller/http/route"
	"consentcc/internal/infra/client/fabric"
	"consentcc/internal/infra/client/postgres"
	"consentcc/internal/infra/client/vault"
	httpSrv "consentcc/internal/infra/server"
	"consentcc/internal/service"
	"github.com/google/wire"
)

type App struct {
	config           *config.Config
	Server           httpSrv.EchoServer
	Vault            vault.IVaultClient
	DBClient         *postgres.Client
	Fabric           fabric.FabricClient
	Consumer         service.IEventConsumer
	healthChainRoute *route.HealthChainRoute
}

var (
	ServerSet                = wire.NewSet(httpSrv.New, wire.Bind(new(httpSrv.EchoServer), new(*httpSrv.Server)))
	VaultClientSet           = wire.NewSet(vault.NewClient, wire.Bind(new(vault.IVaultClient), new(*vault.Client)))
	PostgresClientSet        = wire.NewSet(postgres.NewDBClient) // Placeholder for Postgres client set
	fabricClientSet          = wire.NewSet(fabric.NewClient, wire.Bind(new(fabric.FabricClient), new(*fabric.Client)))
	consentServiceSet        = wire.NewSet(service.NewHealthChainService, wire.Bind(new(service.HealthChainProvider), new(*service.HealthChainService)))
	eventConsumerSet         = wire.NewSet(service.NewEventConsumer, wire.Bind(new(service.IEventConsumer), new(*service.EventConsumer)))
	registerRoutes           = wire.NewSet(route.NewHealthChainRoute)
	healthChainControllerSet = wire.NewSet(
		controller.NewHTTPController,
		wire.Bind(new(controller.IHealthChainController), new(*controller.HealthChainController)),
	)
	ApplicationSet = wire.NewSet(config.Load, NewApp)
)

func NewApp(
	cfg *config.Config,
	serv httpSrv.EchoServer,
	vaultClient vault.IVaultClient,
	db *postgres.Client,
	fabricClient fabric.FabricClient,
	healthChainRoute *route.HealthChainRoute,
	consumer service.IEventConsumer,
) *App {
	return &App{
		Vault:            vaultClient,
		config:           cfg,
		Server:           serv,
		DBClient:         db,
		healthChainRoute: healthChainRoute,
		Fabric:           fabricClient,
		Consumer:         consumer,
	}
}
