package rpc

import (
	"context"
	"fmt"

	"github.com/spooky-finn/go-cryptomarkets-bridge/domain"
	gen "github.com/spooky-finn/go-cryptomarkets-bridge/gen"
)

func (s *server) GetOrderBookSnapshot(ctx context.Context, in *gen.GetOrderBookSnapshotRequest) (*gen.GetOrderBookSnapshotResponse, error) {
	if !s.validationService.IsSupportedProvider(in.Provider) {
		return nil, fmt.Errorf("provider %s is not supported", in.Provider)
	}

	marketSymbol, err := domain.NewMarketSymbolFromString(in.Market)
	if err != nil {
		return nil, fmt.Errorf("invalid market symbol %s. Correct market symbol should use / as a separator", in.Market)
	}

	snapshot, err := s.orderbookSnapshotUseCase.GetOrderBookSnapshot(in.Provider, marketSymbol, int(in.MaxDepth))
	if err != nil {
		return nil, err
	}

	// fmt.Println("snapshot sending to client", snapshot)

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
		Source: selectOrderBookSource(snapshot.Source),
		Bids:   bids,
		Asks:   asks,
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
