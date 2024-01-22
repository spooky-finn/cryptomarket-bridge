package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	pb "github.com/spooky-finn/go-cryptomarkets-bridge/cryptobridge"
	"github.com/spooky-finn/go-cryptomarkets-bridge/domain"
	"github.com/spooky-finn/go-cryptomarkets-bridge/provider"
	"github.com/spooky-finn/go-cryptomarkets-bridge/usecase"

	"google.golang.org/grpc"
)

var (
	port               = flag.Int("port", 50051, "The server port")
	availableProviders = flag.String("providers", "binance,kucoin", "The available providers")
)

func isSupportedProvider(provider string) bool {
	supportedProviders := strings.Split(*availableProviders, ",")
	for _, p := range supportedProviders {
		if p == provider {
			return true
		}
	}
	return false
}

type server struct {
	orderbookSnapshotUseCase *usecase.OrderBookSnapshotUseCase
	pb.UnimplementedMarketDataServiceServer
}

func NewServer() *server {
	apiResolver := provider.NewAPIResolver()

	return &server{
		orderbookSnapshotUseCase: usecase.NewOrderBookSnapshotUseCase(apiResolver),
	}
}

func (s *server) GetOrderBookSnapshot(ctx context.Context, in *pb.GetOrderBookSnapshotRequest) (*pb.GetOrderBookSnapshotResponse, error) {
	if !isSupportedProvider(in.Provider) {
		return nil, fmt.Errorf("provider %s is not supported", in.Provider)
	}

	marketSymbol, err := domain.NewMarketSymbolFromString(in.Market)
	if err != nil {
		return nil, fmt.Errorf("invalid market symbol %s. Correct market symbol should use / as a separator", in.Market)
	}

	maxDepth, err := strconv.ParseInt(in.MaxDepth, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid max depth %s", in.MaxDepth)
	}

	snapshot, err := s.orderbookSnapshotUseCase.GetOrderBookSnapshot(in.Provider, marketSymbol, int(maxDepth))
	if err != nil {
		return nil, err
	}

	// fmt.Println("snapshot sending to client", snapshot)

	bids := []*pb.OrderBookLevel{}
	for _, bid := range snapshot.Bids {
		bids = append(bids, &pb.OrderBookLevel{
			Price: bid[0],
			Qty:   bid[1],
		})
	}

	asks := []*pb.OrderBookLevel{}
	for _, ask := range snapshot.Asks {
		asks = append(asks, &pb.OrderBookLevel{
			Price: ask[0],
			Qty:   ask[1],
		})
	}

	return &pb.GetOrderBookSnapshotResponse{
		Bids: bids,
		Asks: asks,
	}, nil
}

func main() {
	godotenv.Load()

	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterMarketDataServiceServer(s, NewServer())

	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

}
