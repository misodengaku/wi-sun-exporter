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
	ID                 string `json:"id"`
	Password           string `json:"password"`
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
	config.ID = os.Getenv("ID")
	if config.ID == "" {
		panic("please specify ID environment variable")
	}
	config.Password = os.Getenv("PASSWORD")
	if config.Password == "" {
		panic("please specify PASSWORD environment variable")
	}

	powerGauge := promauto.NewGauge(prometheus.GaugeOpts{
		Name: "route_b_instant_power",
		Help: "Instantaneous power value",
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
	log.Println("connecting...")
	err = device.SetAuthentication(config.ID, config.Password)
	if err != nil {
		panic(err)
	}
	channelScanResult, err := device.ChannelScan(6)
	if err != nil {
		panic(err)
	}
	device.SetChannel(channelScanResult.Channel)
	device.SetPanID(channelScanResult.PanID)
	device.ExecutePANAAuth(channelScanResult.IPv6Address)
	err = device.WaitForPANAAuth()
	if err != nil {
		panic(err)
	}
	log.Println("wi-sun-exporter is running")

	for {
		power, err := device.GetInstantPower(channelScanResult.IPv6Address)
		if err != nil {
			panic(err)
		}
		powerGauge.Set(float64(power))
		time.Sleep(15 * time.Second)
	}
}
