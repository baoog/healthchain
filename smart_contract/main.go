package main

import (
	"consentcc/controller"
	"consentcc/service"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"log"
)

func main() {
	consentService := service.NewConsentService()
	consentController := controller.NewConsentService(consentService)
	cc, err := contractapi.NewChaincode(consentController)
	if err != nil {
		panic(err)
	}

	log.Default().Println("Starting ConsentCC Smart Contract")
	err = cc.Start()
	if err != nil {
		panic(err)
	}
	log.Println("ConsentCC Smart Contract started successfully")
}
