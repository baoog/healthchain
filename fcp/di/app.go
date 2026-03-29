package di

import (
	"fcp/config"
	"fcp/internal/adapter/repository"
	httpController "fcp/internal/controller/http"
	"fcp/internal/controller/http/route"
	"fcp/internal/infra/client/fabric"
	"fcp/internal/infra/client/jwt"
	"fcp/internal/infra/client/postgres"
	httpSrv "fcp/internal/infra/server"
	"fcp/internal/service"
	"github.com/google/wire"
)

type App struct {
	config          *config.Config
	Server          httpSrv.EchoServer
	DBClient        *postgres.Client
	federationCntl  *httpController.FederationHandler
	client          fabric.FabricClient
	federationRoute *route.FcpRoute
}

var (
	ServerSet                = wire.NewSet(httpSrv.New, wire.Bind(new(httpSrv.EchoServer), new(*httpSrv.Server)))
	PostgresClientSet        = wire.NewSet(postgres.NewDBClient) // Placeholder for Postgres client set
	fabricClientSet          = wire.NewSet(fabric.NewClient, wire.Bind(new(fabric.FabricClient), new(*fabric.Client)))
	jwtClientSet             = wire.NewSet(jwt.NewJWTClient, wire.Bind(new(jwt.JWTClientProvider), new(*jwt.JWTClient)))
	hospitalRepoSet          = wire.NewSet(repository.NewHospitalRepo)            // Placeholder for Hospital repository set
	HospitalEndpointRepoSet  = wire.NewSet(repository.NewEndpointRepo)            // Placeholder for Hospital Endpoint repository set
	keyRepoSet               = wire.NewSet(repository.NewKeyRepo)                 // Placeholder for Key repository set
	hospitalInterfaceRepoSet = wire.NewSet(repository.NewFederationInterfaceRepo) // Placeholder for Federation Interface repository set
	federationServiceSet     = wire.NewSet(                                       // Placeholder for Federation service set
		service.NewFederationService,
	)
	federationRouteSet      = wire.NewSet(route.NewFcpRoute)
	federationControllerSet = wire.NewSet(httpController.NewFederationHandler)
	ApplicationSet          = wire.NewSet(config.Load, NewApp)
)

func NewApp(
	cfg *config.Config,
	serv httpSrv.EchoServer,
	db *postgres.Client,
	federationCntl *httpController.FederationHandler,
	client fabric.FabricClient,
	federationRoute *route.FcpRoute,
) *App {
	return &App{
		config:          cfg,
		Server:          serv,
		DBClient:        db,
		federationCntl:  federationCntl,
		client:          client,
		federationRoute: federationRoute,
	}
}
