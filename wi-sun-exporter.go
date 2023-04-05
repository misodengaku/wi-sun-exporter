package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/misodengaku/wi-sun-exporter/mbrl7023"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Config struct {
	PromHTTPListenAddr string `json:"prometheus_listen_addr"`
	TTY                string `json:"tty"`
}

func main() {
	var err error
	var config Config
	config.PromHTTPListenAddr = os.Getenv("LISTEN_ADDR")
	if config.PromHTTPListenAddr == "" {
		panic("please specify LISTEN_ADDR environment variable")
	}
	config.TTY = os.Getenv("TTY")
	if config.TTY == "" {
		panic("please specify TTY environment variable")
	}

	co2Gauge := promauto.NewGauge(prometheus.GaugeOpts{
		Name: "udco2s_co2_concentration",
		Help: "CO2 concentration",
	})
	humGauge := promauto.NewGauge(prometheus.GaugeOpts{
		Name: "udco2s_humidity",
		Help: "Humidity",
	})
	tempGauge := promauto.NewGauge(prometheus.GaugeOpts{
		Name: "udco2s_temperature",
		Help: "Temperature",
	})

	go func() {
		// promhttp
		http.Handle("/metrics", promhttp.Handler())
		log.Println(http.ListenAndServe(config.PromHTTPListenAddr, nil))
	}()

	device := mbrl7023.MBRL7023{}
	err = device.Init(context.Background(), config.TTY)
	if err != nil {
		panic(err)
	}
	device.StartMeasurement()
	log.Println("wi-sun-exporter is running")

	for {
		co2Gauge.Set(float64(device.CO2))
		humGauge.Set(device.Humidity)
		tempGauge.Set(device.Temperature)
		time.Sleep(1 * time.Second)
	}
}
