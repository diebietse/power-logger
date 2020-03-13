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
	// VoltageReg input voltage register of 16 bits
	VoltageReg = 0
	// CurrentReg input current register of 16 bits
	CurrentReg = 2
	// FrequencyReg input frequency register of 16 bits
	FrequencyReg = 4
	// ActivePowerReg input active power register of 16 bits
	ActivePowerReg = 6
	// ReactivePowerReg input reactive power register of 16 bits
	ReactivePowerReg = 8
	// ApparentPowerReg input apparent power register of 16 bits
	ApparentPowerReg = 10
	// PowerFactorReg input power factor register of 16 bits
	PowerFactorReg = 12
	// ActiveEnergyReg input active energy register of 5 x 32 bits
	ActiveEnergyReg = 14
	// ReactiveEnergyReg input reactive energy register of 5 x 32 bits
	ReactiveEnergyReg = 34
	// TsReg energy time slot registers of 4 x 24 bits
	TsReg = 54
	// TimeReg internal real time clock for the time slots at 64 bits
	TimeReg = 66
	// TemperatureReg device temperature register of 16 bits
	TemperatureReg = 74
)

const (
	readSize    = 39
	pollRateSec = 10
)

// Logger contains the Gauges for a logger instance
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
	sticky    bool
}

// New returns new logger with a given name and modbus client
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
			register:  VoltageReg,
			scale:     10,
			valueFunc: get16BitValue,
		},
		{
			Gauge: prometheus.NewGauge(prometheus.GaugeOpts{
				Name:        "mains_current_a",
				Help:        "Mains current",
				ConstLabels: label,
			}),
			register:  CurrentReg,
			scale:     10,
			valueFunc: get16BitValue,
		},
		{
			Gauge: prometheus.NewGauge(prometheus.GaugeOpts{
				Name:        "mains_frequency_hz",
				Help:        "Mains frequency",
				ConstLabels: label,
			}),
			register:  FrequencyReg,
			scale:     10,
			valueFunc: get16BitValue,
		},
		{
			Gauge: prometheus.NewGauge(prometheus.GaugeOpts{
				Name:        "mains_active_power_w",
				Help:        "Mains active power",
				ConstLabels: label,
			}),
			register:  ActivePowerReg,
			scale:     1,
			valueFunc: get16BitValue,
		},
		{
			Gauge: prometheus.NewGauge(prometheus.GaugeOpts{
				Name:        "mains_reactive_power_var",
				Help:        "Mains reactive power",
				ConstLabels: label,
			}),
			register:  ReactivePowerReg,
			scale:     1,
			valueFunc: get16BitValue,
		},
		{
			Gauge: prometheus.NewGauge(prometheus.GaugeOpts{
				Name:        "mains_appartent_power_va",
				Help:        "Mains appartent power",
				ConstLabels: label,
			}),
			register:  ApparentPowerReg,
			scale:     1,
			valueFunc: get16BitValue,
		},
		{
			Gauge: prometheus.NewGauge(prometheus.GaugeOpts{
				Name:        "mains_power_factor_pf",
				Help:        "Mains power factor",
				ConstLabels: label,
			}),
			register:  PowerFactorReg,
			scale:     1000,
			valueFunc: get16BitValue,
		},
		{
			Gauge: prometheus.NewGauge(prometheus.GaugeOpts{
				Name:        "mains_active_energy_kwh",
				Help:        "Mains active energy",
				ConstLabels: label,
			}),
			register:  ActiveEnergyReg,
			scale:     100,
			valueFunc: get32BitEnergy,
			sticky:    true,
		},
		{
			Gauge: prometheus.NewGauge(prometheus.GaugeOpts{
				Name:        "mains_reactive_energy_kvarh",
				Help:        "Mains reactive energy",
				ConstLabels: label,
			}),
			register:  ReactiveEnergyReg,
			scale:     100,
			valueFunc: get32BitEnergy,
			sticky:    true,
		},
		{
			Gauge: prometheus.NewGauge(prometheus.GaugeOpts{
				Name:        "mains_device_temperature_c",
				Help:        "Mains device temperature",
				ConstLabels: label,
			}),
			register:  TemperatureReg,
			scale:     1,
			valueFunc: get16BitValue,
		},
	}
}

func (l *Logger) update() error {
	res, err := l.client.ReadHoldingRegisters(0, readSize)
	if err != nil {
		l.errorEvent()
		return fmt.Errorf("could not read values: %v", err)
	}
	if len(res) != readSize*2 {
		l.errorEvent()
		return fmt.Errorf("invalid read size: %v", len(res))
	}

	for _, g := range l.gauges {
		g.Set(g.valueFunc(res, g.register, g.scale))
	}
	return nil
}

func (l *Logger) errorEvent() {
	l.readFailures.Add(1)
	for _, g := range l.gauges {
		if !g.sticky {
			g.Set(0)
		}
	}
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
