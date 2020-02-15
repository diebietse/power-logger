package main

import (
	"encoding/binary"
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/goburrow/modbus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type gauges struct {
	client         modbus.Client
	voltage        prometheus.Gauge
	current        prometheus.Gauge
	frequency      prometheus.Gauge
	activePower    prometheus.Gauge
	reactivePower  prometheus.Gauge
	appartentPower prometheus.Gauge
	powerFactor    prometheus.Gauge
	readFailures   prometheus.Gauge
	activeEnergy   prometheus.Gauge
	reactiveEnergy prometheus.Gauge
	temperature    prometheus.Gauge
}

const (
	voltageReg        = 0
	currentReg        = 2
	frequencyReg      = 4
	activePowerReg    = 6
	reactivePowerReg  = 8
	apparentPowerReg  = 10
	powerFactorReg    = 12
	activeEnergyReg   = 14
	reactiveEnergyReg = 34
	tsReg             = 54
	timeReg           = 66
	tempReg           = 74
)

func newGauge(c modbus.Client) *gauges {
	g := &gauges{
		client: c,
		voltage: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "mains_voltage_v",
			Help: "Mains voltage",
		}),
		current: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "mains_current_a",
			Help: "Mains current",
		}),
		frequency: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "mains_frequency_v",
			Help: "Mains frequency",
		}),
		activePower: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "mains_active_power_w",
			Help: "Mains active power",
		}),
		reactivePower: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "mains_reactive_power_var",
			Help: "Mains reactive power",
		}),
		appartentPower: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "mains_appartent_power_va",
			Help: "Mains appartent power",
		}),
		powerFactor: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "mains_power_factor_pf",
			Help: "Mains power factor",
		}),
		readFailures: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "sensor_read_errors_count",
			Help: "Sensor read errors",
		}),
		activeEnergy: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "mains_active_energy_kwh",
			Help: "Mains active energy",
		}),
		reactiveEnergy: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "mains_reactive_energy_kvarh",
			Help: "Mains reactive energy",
		}),
		temperature: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "mains_device_temperature_c",
			Help: "Mains device temperature",
		}),
	}

	prometheus.MustRegister(
		g.voltage,
		g.current,
		g.frequency,
		g.activePower,
		g.reactivePower,
		g.appartentPower,
		g.powerFactor,
		g.readFailures,
		g.activeEnergy,
		g.reactiveEnergy,
		g.temperature,
	)
	return g
}

func getValue(data []byte, offset int, scale float64) float64 {
	return float64(binary.BigEndian.Uint16(data[offset:offset+2])) / scale
}

func (g *gauges) update() {
	readLength := 39
	res, err := g.client.ReadHoldingRegisters(0, 39)
	if err != nil {
		log.Printf("Error updating values: %v", err)
		g.readFailures.Add(1)
		return
	}
	if len(res) != readLength*2 {
		log.Printf("Invalid result length: %v", len(res))
		g.readFailures.Add(1)
		return
	}
	g.voltage.Set(getValue(res, voltageReg, 10))
	g.current.Set(getValue(res, currentReg, 10))
	g.frequency.Set(getValue(res, frequencyReg, 10))
	g.activePower.Set(getValue(res, activePowerReg, 1))
	g.reactivePower.Set(getValue(res, reactivePowerReg, 1))
	g.appartentPower.Set(getValue(res, apparentPowerReg, 1))
	g.powerFactor.Set(getValue(res, powerFactorReg, 1000))
	g.activeEnergy.Set(getEnergy(res, activeEnergyReg))
	g.reactiveEnergy.Set(getEnergy(res, reactiveEnergyReg))
	g.temperature.Set(getValue(res, tempReg, 1))
}

func getEnergy(data []byte, offset int) float64 {
	return float64(binary.BigEndian.Uint32(data[offset:offset+4])) / 100
}

func printEnergy(data []byte, offset int, name string) {
	t := binary.BigEndian.Uint32(data[offset : offset+4])
	log.Printf("T (%s): %x %v", name, data[offset:offset+4], t)

	offset = offset + 4
	t1 := binary.BigEndian.Uint32(data[offset : offset+4])
	log.Printf("T1 (%s): %x %v", name, data[offset:offset+4], t1)

	offset = offset + 4
	t2 := binary.BigEndian.Uint32(data[offset : offset+4])
	log.Printf("T2 (%s): %x %v", name, data[offset:offset+4], t2)

	offset = offset + 4
	t3 := binary.BigEndian.Uint32(data[offset : offset+4])
	log.Printf("T3 (%s): %x %v", name, data[offset:offset+4], t3)

	offset = offset + 4
	t4 := binary.BigEndian.Uint32(data[offset : offset+4])
	log.Printf("T4 (%s): %x %v", name, data[offset:offset+4], t4)
}

func printTS(data []byte, offset int) {
	log.Printf("%.2x-%.2x-%.2x", data[offset+0], data[offset+1], data[offset+2])
	log.Printf("%.2x-%.2x-%.2x", data[offset+3], data[offset+4], data[offset+5])
	log.Printf("%.2x-%.2x-%.2x", data[offset+6], data[offset+7], data[offset+8])
	log.Printf("%.2x-%.2x-%.2x", data[offset+9], data[offset+10], data[offset+11])
}

func printTime(data []byte, offset int) {
	log.Printf("time: %.2x%.2x-%.2x-%.2x %.2x %.2x:%.2x:%.2x", data[offset+0], data[offset+1], data[offset+2], data[offset+3], data[offset+4], data[offset+5], data[offset+6], data[offset+7])
	t := binary.BigEndian.Uint64(data[offset : offset+8])
	log.Printf("64bit: %x %d", t, t)
}

func printTemp(data []byte, offset int) {
	log.Printf("Temperature: %v", getValue(data, offset, 1))
}

func main() {
	addr := flag.String("addr", ":8080", "TCP address to listen on.")
	dev := flag.String("dev", "/dev/ttyS0", "TTY device to use.")
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
		panic(err)
	}
	defer handler.Close()

	http.Handle("/metrics", promhttp.Handler())

	client := modbus.NewClient(handler)
	log.Printf("Connected to device: %v", *dev)

	g := newGauge(client)
	go func() {
		ticker := time.NewTicker(time.Second * 10)
		for _ = range ticker.C {
			g.update()
		}
	}()

	log.Printf("Starting server: %v", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
