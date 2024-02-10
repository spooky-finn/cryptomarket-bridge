package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spooky-finn/go-cryptomarkets-bridge/config"
	gen "github.com/spooky-finn/go-cryptomarkets-bridge/gen"
	promclient "github.com/spooky-finn/go-cryptomarkets-bridge/infrastructure/prometheus"
	"github.com/spooky-finn/go-cryptomarkets-bridge/rpc"
	"google.golang.org/grpc"
)

var (
	port               = flag.Int("port", 50051, "The server port")
	availableProviders = flag.String("providers", "binance,kucoin", "The available providers")
	debugMode          = flag.Bool("v", false, "Enable debug mode")
)

func main() {
	godotenv.Load()
	flag.Parse()
	go promclient.StartPromClientServer()

	config.DebugMode = *debugMode
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

	// syncAPI := kucoin.NewKucoinSyncAPI()
	// token, err := syncAPI.WsConnOpts()
	// if err != nil {
	// 	fmt.Printf("Error while creating ws connection options %s", err.Error())
	// 	panic(err)
	// }
	// kucoinStreamClien := kucoin.NewKucoinStreamClient(token)
	// _, _, err = kucoinStreamClien.Connect()
	// if err != nil {
	// 	fmt.Printf("Error while connecting to kucoin %s", err.Error())
	// 	panic(err)
	// }

	// streamAPI := kucoin.NewKucoinStreamAPI(kucoinStreamClien, syncAPI)
	// s, err := domain.NewMarketSymbolFromString("BTC_USDT")
	// if err != nil {
	// 	fmt.Printf("Error while creating market symbol %s", err.Error())
	// 	panic(err)
	// }
	// subscribtion, err := streamAPI.DepthDiffStream(s)
	// if err != nil {
	// 	fmt.Printf("Error while subscribing to kucoin %s", err.Error())
	// 	panic(err)
	// }

	// for {
	// 	select {
	// 	case msg := <-subscribtion.Stream:
	// 		fmt.Println("depth strea: ", string(msg.Symbol))
	// 	}
	// }
}
