package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/misodengaku/wi-sun-exporter/mbrl7023"
)

type Config struct {
	ID       string `json:"id"`
	Password string `json:"password"`
	TTY      string `json:"tty"`
}

func main() {
	config := Config{}
	configBytes, err := os.ReadFile("config.json")
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(configBytes, &config)
	if err != nil {
		panic(err)
	}

	fmt.Println("hello")
	device := mbrl7023.MBRL7023{}
	println("init")
	device.Init(context.Background(), config.TTY)
	println("auth")
	device.SetAuthentication(config.ID, config.Password)
	println("scan")
	channelScanResult, err := device.ChannelScan(6)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%#v\n", channelScanResult)
	channelBytes, err := json.Marshal(channelScanResult)
	if err != nil {
		panic(err)
	}
	err = os.WriteFile("pandesc.json", channelBytes, 0644)
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
	println("joined")

	for {
		power, err := device.GetInstantPower(channelScanResult.IPv6Address)
		if err != nil {
			panic(err)
		}
		fmt.Printf("power: %d[W]\n", power)
		time.Sleep(30 * time.Second)
	}
}
