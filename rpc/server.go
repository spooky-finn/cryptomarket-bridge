package rpc

import (
	gen "github.com/spooky-finn/go-cryptomarkets-bridge/gen"
	"github.com/spooky-finn/go-cryptomarkets-bridge/provider"
	"github.com/spooky-finn/go-cryptomarkets-bridge/usecase"
)

type server struct {
	orderbookSnapshotUseCase *usecase.OrderBookSnapshotUseCase
	gen.UnimplementedMarketDataServiceServer
	validationService *ValidationService
}

func NewServer(conf *ValidationServiceConfig) *server {
	apiResolver := provider.NewAPIResolver()

	return &server{
		orderbookSnapshotUseCase: usecase.NewOrderBookSnapshotUseCase(apiResolver),
		validationService:        NewValidationService(conf),
	}
}
