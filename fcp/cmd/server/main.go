package main

import (
	"context"
	"fcp/di"
	"fcp/internal/infra/server"
)

func main() {
	app, _, err := di.InitializeApplication()
	if err != nil {
		panic(err)
	}
	app.Server.StartAsync()
	server.GracefulWait(context.Background(), func() {})
}
