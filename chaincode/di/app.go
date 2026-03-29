package di

import (
	"consentcc/config"
	"consentcc/internal/adapter/repository"
	controller "consentcc/internal/controller/http"
	"consentcc/internal/controller/http/route"
	"consentcc/internal/infra/client/fabric"
	"consentcc/internal/infra/client/fcp"
	"consentcc/internal/infra/client/jwt"
	"consentcc/internal/infra/client/minio"
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
	ServerSet              = wire.NewSet(httpSrv.New, wire.Bind(new(httpSrv.EchoServer), new(*httpSrv.Server)))
	VaultClientSet         = wire.NewSet(vault.NewClient, wire.Bind(new(vault.IVaultClient), new(*vault.Client)))
	PostgresClientSet      = wire.NewSet(postgres.NewDBClient) // Placeholder for Postgres client set
	fabricClientSet        = wire.NewSet(fabric.NewClient, wire.Bind(new(fabric.FabricClient), new(*fabric.Client)))
	jwtClientSet           = wire.NewSet(jwt.NewJWTClient, wire.Bind(new(jwt.JWTClientProvider), new(*jwt.JWTClient)))
	FcpClientSet           = wire.NewSet(fcp.NewClient)
	bundleRepoSet          = wire.NewSet(repository.NewBundleRepo, wire.Bind(new(repository.BundleRepoProvider), new(*repository.BundleRepo)))
	consentRepoSet         = wire.NewSet(repository.NewConsentRepo, wire.Bind(new(repository.ConsentRepoProvider), new(*repository.ConsentRepo)))
	emergencyAccessRepoSet = wire.NewSet(repository.NewEmergencyAccessRepo, wire.Bind(new(repository.EmergencyAccessRepoProvider), new(*repository.EmergencyAccessRepo)))
	userRepoSet            = wire.NewSet(repository.NewUserRepo, wire.Bind(new(repository.UserRepoProvider), new(*repository.UserRepo)))

	consentServiceSet   = wire.NewSet(service.NewHealthChainService, wire.Bind(new(service.HealthChainProvider), new(*service.HealthChainService)))
	orchestratorService = wire.NewSet(service.NewOrchestratorService, wire.Bind(new(service.OrchestratorServiceProvider), new(*service.OrchestratorService)))
	eventConsumerSet    = wire.NewSet(service.NewEventConsumer, wire.Bind(new(service.IEventConsumer), new(*service.EventConsumer)))
	userServiceSet      = wire.NewSet(service.NewUserService, wire.Bind(new(service.UserServiceProvider), new(*service.UserService)))

	registerRoutes           = wire.NewSet(route.NewHealthChainRoute)
	healthChainControllerSet = wire.NewSet(
		controller.NewHTTPController,
		wire.Bind(new(controller.IHealthChainController), new(*controller.HealthChainController)),
	)
	orchestratorControllerSet = wire.NewSet(
		controller.NewOrchestratorHTTPController,
		wire.Bind(new(controller.IOrchestratorController), new(*controller.OrchestratorController)),
	)
	minioClientSet = wire.NewSet(minio.NewClient, wire.Bind(new(minio.IMinoIOClient), new(*minio.Client))) // Placeholder for MinIO client set
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
