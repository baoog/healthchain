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
		consentServiceSet,
		eventConsumerSet,
		healthChainControllerSet,
		registerRoutes,
		ApplicationSet,
	)
	return new(App), nil, nil
}
