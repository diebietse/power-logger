package main

import (
	"flag"
	"net/http"
	"time"

	"github.com/diebietse/power-logger/logger"
	"github.com/goburrow/modbus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

func main() {
	addr := flag.String("addr", ":8080", "TCP address to listen on.")
	dev := flag.String("dev", "/dev/ttyS0", "TTY device to use.")
	deviceName := flag.String("deviceName", "flat-power", "Set the device_name label.")
	flag.Parse()

	// Modbus RTU/ASCII
	handler := modbus.NewRTUClientHandler(*dev)
	handler.BaudRate = 9600
	handler.DataBits = 8
	handler.Parity = "N"
	handler.StopBits = 1
	handler.SlaveId = 1
	handler.Timeout = 5 * time.Second

	err := handler.Connect()
	if err != nil {
		log.Fatal(err)
	}
	defer handler.Close()

	http.Handle("/metrics", promhttp.Handler())

	client := modbus.NewClient(handler)
	l, err := logger.New(client, *deviceName)
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()
	l.Poller()

	log.Printf("Starting server: %v", *addr)
	err = http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatal(err)
	}
}
