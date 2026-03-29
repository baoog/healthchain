//go:build wireinject
// +build wireinject

package di

import (
	"github.com/google/wire"
)

func InitializeApplication() (*App, func(), error) {
	wire.Build(
		ServerSet,
		PostgresClientSet,
		fabricClientSet,
		//jwtClientSet,
		hospitalRepoSet,
		HospitalEndpointRepoSet,
		keyRepoSet,
		hospitalInterfaceRepoSet,
		federationServiceSet,
		federationControllerSet,
		federationRouteSet,
		ApplicationSet,
	)
	return new(App), nil, nil
}
