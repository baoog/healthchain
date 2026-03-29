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
		VaultClientSet,
		fabricClientSet,
		jwtClientSet,
		FcpClientSet,
		bundleRepoSet,
		consentRepoSet,
		userRepoSet,
		consentServiceSet,
		emergencyAccessRepoSet,
		orchestratorService,
		userServiceSet,
		eventConsumerSet,
		healthChainControllerSet,
		orchestratorControllerSet,
		registerRoutes,
		minioClientSet,
		ApplicationSet,
	)
	return new(App), nil, nil
}
