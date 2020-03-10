# power-logger

Go application to log electrical power of the YTL-e D113003 to a Prometheus endpoint.

## Quick Start

For RaspberryPi build run

```
make build-arm
```

For x64 builds run

```
make build-x64
```

## Usage

```
Usage of ./power-logger:
  -addr string
        TCP address to listen on for prometheus. (default ":8080")
  -dev string
        TTY device to use. (default "/dev/ttyS0")
  -deviceName string
        Set the device_name label in prometheus. (default "flat-power")
```

[build-status]: https://github.com/ncthompson/power-logger//workflows/build/badge.svg?branch=master