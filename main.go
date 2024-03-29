package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spooky-finn/cryptobridge/config"
	gen "github.com/spooky-finn/cryptobridge/gen"
	promclient "github.com/spooky-finn/cryptobridge/infrastructure/prometheus"
	"github.com/spooky-finn/cryptobridge/rpc"
	"google.golang.org/grpc"
)

var (
	port                       = flag.Int("port", 50051, "The server port")
	availableProviders         = flag.String("providers", "binance,kucoin", "The available providers")
	debugMode                  = flag.Bool("v", false, "Enable debug mode")
	orderBookMaxSupportedDepth = flag.Int("max-orderbook-depth", 1000, "The maximum rows in the orderbook to guaranelly be served")
)

func main() {
	godotenv.Load()
	flag.Parse()
	go promclient.StartPromClientServer()

	config.DebugMode = *debugMode
	config.OrderBookMaxSupportedDepth = *orderBookMaxSupportedDepth

	if config.DebugMode {
		log.Println("Debug mode enabled")
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	conf := &rpc.ValidationServiceConfig{
		AvailableProviders: strings.Split(*availableProviders, ","),
	}
	gen.RegisterMarketDataServiceServer(s, rpc.NewServer(conf))

	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
