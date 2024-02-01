package promclient

import (
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var BinanceOpenOrderBookGauge = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "binance_open_order_book",
		Help: "binance open order book",
	},
)

var KucoinOpenOrderBookGauge = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "kucoin_open_order_book",
		Help: "kucoin open order book",
	},
)

func StartPromClientServer() {
	reg := prometheus.NewRegistry()
	promHnadler := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})

	reg.MustRegister(BinanceOpenOrderBookGauge)
	reg.MustRegister(KucoinOpenOrderBookGauge)
	reg.MustRegister(collectors.NewGoCollector())

	http.Handle("/metrics", promHnadler)
	log.Printf("prometheus server listening at :8080")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
