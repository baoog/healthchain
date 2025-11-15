package main

import (
	"consentcc/di"
	"consentcc/internal/infra/server"
	"context"
)

func main() {
	// This is a placeholder main function.
	//chainc, err := contractapi.NewChaincode(new(entity.SmartContract))
	//if err != nil {
	//	panic(err.Error())
	//}
	//if err := chainc.Start(); err != nil {
	//	panic(err.Error())
	//}
	app, _, err := di.InitializeApplication()
	if err != nil {
		panic(err)
	}
	app.Consumer.Start(context.Background())
	app.Server.StartAsync()
	server.GracefulWait(context.Background(), func() {})
}
