package rpc

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/spooky-finn/go-cryptomarkets-bridge/domain"
	gen "github.com/spooky-finn/go-cryptomarkets-bridge/gen"
)

var logger = log.New(os.Stdout, "rpc: ", log.LstdFlags)

func (s *server) GetOrderBookSnapshot(ctx context.Context, in *gen.GetOrderBookSnapshotRequest) (*gen.GetOrderBookSnapshotResponse, error) {
	if !s.validationService.IsSupportedProvider(in.Provider) {
		return nil, fmt.Errorf("provider %s is not supported", in.Provider)
	}

	marketSymbol, err := domain.NewMarketSymbolFromString(in.Market)
	if err != nil {
		logger.Printf("error parsing market symbol: %s", err)
		return nil, fmt.Errorf("invalid market symbol %s. Correct market symbol should use _ as a separator", in.Market)
	}

	snapshot, err := s.orderbookSnapshotUseCase.GetOrderBookSnapshot(in.Provider, marketSymbol, int(in.MaxDepth))
	if err != nil {
		logger.Printf("error getting order book snapshot: %s", err)
		return nil, err
	}

	bids := []*gen.OrderBookLevel{}
	for _, bid := range snapshot.Bids {
		bids = append(bids, &gen.OrderBookLevel{
			Price: bid[0],
			Qty:   bid[1],
		})
	}

	asks := []*gen.OrderBookLevel{}
	for _, ask := range snapshot.Asks {
		asks = append(asks, &gen.OrderBookLevel{
			Price: ask[0],
			Qty:   ask[1],
		})
	}

	return &gen.GetOrderBookSnapshotResponse{
		LastUpdTs: snapshot.LastUpdateId,
		Source:    selectOrderBookSource(snapshot.Source),
		Bids:      bids,
		Asks:      asks,
	}, nil
}

func selectOrderBookSource(source domain.OrderBookSource) gen.OrderBookSource {
	switch source {
	case domain.OrderBookSource_LocalOrderBook:
		return gen.OrderBookSource_LocalOrderBook
	case domain.OrderBookSource_Provider:
		return gen.OrderBookSource_Provider
	default:
		return gen.OrderBookSource_Unknown
	}
}
