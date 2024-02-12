package rpc

import (
	gen "github.com/spooky-finn/cryptobridge/gen"
	"github.com/spooky-finn/cryptobridge/provider"
	"github.com/spooky-finn/cryptobridge/usecase"
)

type server struct {
	orderbookSnapshotUseCase *usecase.OrderBookSnapshotUseCase
	gen.UnimplementedMarketDataServiceServer
	validationService *ValidationService
}

func NewServer(conf *ValidationServiceConfig) *server {
	connManager := provider.NewConnectionManager()
	connManager.Init()

	return &server{
		orderbookSnapshotUseCase: usecase.NewOrderBookSnapshotUseCase(connManager),
		validationService:        NewValidationService(conf),
	}
}
