package main

import (
	"consentcc/di"
	"consentcc/internal/infra/server"
	"context"
)

func main() {
	app, _, err := di.InitializeApplication()
	if err != nil {
		panic(err)
	}
	app.Consumer.Start(context.Background())
	app.Server.StartAsync()
	server.GracefulWait(context.Background(), func() {})
}
