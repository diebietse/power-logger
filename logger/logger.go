package logger

import (
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	"github.com/goburrow/modbus"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

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
	temperatureReg    = 74
)

const (
	readSize    = 39
	pollRateSec = 10
)

type Logger struct {
	client       modbus.Client
	gauges       []loggerGauge
	readFailures prometheus.Gauge
	wg           sync.WaitGroup
	stop         chan struct{}
}

type loggerGauge struct {
	prometheus.Gauge
	register  int
	scale     float64
	valueFunc func(data []byte, offset int, scale float64) float64
}

func New(client modbus.Client, deviceName string) (*Logger, error) {
	label := map[string]string{"device_name": deviceName}

	l := &Logger{
		client: client,
		gauges: generateGauges(label),
		readFailures: prometheus.NewGauge(prometheus.GaugeOpts{
			Name:        "sensor_read_errors_count",
			Help:        "Sensor read errors",
			ConstLabels: label,
		}),
		wg:   sync.WaitGroup{},
		stop: make(chan struct{}),
	}

	for _, g := range l.gauges {
		if err := prometheus.Register(g); err != nil {
			return nil, fmt.Errorf("could not register gauge: %v", err)
		}
	}

	if err := prometheus.Register(l.readFailures); err != nil {
		return nil, fmt.Errorf("could not register gauge: %v", err)
	}

	return l, nil
}

func generateGauges(label map[string]string) []loggerGauge {
	return []loggerGauge{
		{
			Gauge: prometheus.NewGauge(prometheus.GaugeOpts{
				Name:        "mains_voltage_v",
				Help:        "Mains voltage",
				ConstLabels: label,
			}),
			register:  voltageReg,
			scale:     10,
			valueFunc: get16BitValue,
		},
		{
			Gauge: prometheus.NewGauge(prometheus.GaugeOpts{
				Name:        "mains_current_a",
				Help:        "Mains current",
				ConstLabels: label,
			}),
			register:  currentReg,
			scale:     10,
			valueFunc: get16BitValue,
		},
		{
			Gauge: prometheus.NewGauge(prometheus.GaugeOpts{
				Name:        "mains_frequency_v",
				Help:        "Mains frequency",
				ConstLabels: label,
			}),
			register:  frequencyReg,
			scale:     10,
			valueFunc: get16BitValue,
		},
		{
			Gauge: prometheus.NewGauge(prometheus.GaugeOpts{
				Name:        "mains_active_power_w",
				Help:        "Mains active power",
				ConstLabels: label,
			}),
			register:  activePowerReg,
			scale:     1,
			valueFunc: get16BitValue,
		},
		{
			Gauge: prometheus.NewGauge(prometheus.GaugeOpts{
				Name:        "mains_reactive_power_var",
				Help:        "Mains reactive power",
				ConstLabels: label,
			}),
			register:  reactivePowerReg,
			scale:     1,
			valueFunc: get16BitValue,
		},
		{
			Gauge: prometheus.NewGauge(prometheus.GaugeOpts{
				Name:        "mains_appartent_power_va",
				Help:        "Mains appartent power",
				ConstLabels: label,
			}),
			register:  apparentPowerReg,
			scale:     1,
			valueFunc: get16BitValue,
		},
		{
			Gauge: prometheus.NewGauge(prometheus.GaugeOpts{
				Name:        "mains_power_factor_pf",
				Help:        "Mains power factor",
				ConstLabels: label,
			}),
			register:  powerFactorReg,
			scale:     1000,
			valueFunc: get16BitValue,
		},
		{
			Gauge: prometheus.NewGauge(prometheus.GaugeOpts{
				Name:        "mains_active_energy_kwh",
				Help:        "Mains active energy",
				ConstLabels: label,
			}),
			register:  activeEnergyReg,
			scale:     100,
			valueFunc: get32BitEnergy,
		},
		{
			Gauge: prometheus.NewGauge(prometheus.GaugeOpts{
				Name:        "mains_reactive_energy_kvarh",
				Help:        "Mains reactive energy",
				ConstLabels: label,
			}),
			register:  reactiveEnergyReg,
			scale:     100,
			valueFunc: get32BitEnergy,
		},
		{
			Gauge: prometheus.NewGauge(prometheus.GaugeOpts{
				Name:        "mains_device_temperature_c",
				Help:        "Mains device temperature",
				ConstLabels: label,
			}),
			register:  temperatureReg,
			scale:     1,
			valueFunc: get16BitValue,
		},
	}
}

func (l *Logger) update() error {
	res, err := l.client.ReadHoldingRegisters(0, readSize)
	if err != nil {
		l.readFailures.Add(1)
		return fmt.Errorf("could not read values: %v", err)
	}
	if len(res) != readSize*2 {
		l.readFailures.Add(1)
		return fmt.Errorf("invalid read size: %v", len(res))
	}

	for _, g := range l.gauges {
		g.Set(g.valueFunc(res, g.register, g.scale))
	}
	return nil
}

// Poller starts the polling of the new values device
func (l *Logger) Poller() {
	l.wg.Add(1)
	defer l.wg.Done()
	ticker := time.NewTicker(time.Second * pollRateSec)
	if err := l.update(); err != nil {
		log.Errorf("Could not update values: %v", err)
	}
	go func() {
		for {
			select {
			case <-ticker.C:
				if err := l.update(); err != nil {
					log.Errorf("Could not update values: %v", err)
				}
			case <-l.stop:
				ticker.Stop()
				return
			}
		}
	}()
}

// Close stops the poller
func (l *Logger) Close() {
	close(l.stop)
	l.wg.Wait()
}

func get16BitValue(data []byte, offset int, scale float64) float64 {
	return float64(binary.BigEndian.Uint16(data[offset:offset+2])) / scale
}

func get32BitEnergy(data []byte, offset int, scale float64) float64 {
	// The time binned data is ignored as the internal clock is never set
	// The layout for the energy mapping is 5 x 32 Big Endian Numbers
	return float64(binary.BigEndian.Uint32(data[offset:offset+4])) / scale
}
