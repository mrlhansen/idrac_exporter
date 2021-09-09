# iDRAC Exporter
This is a (very) simple iDRAC exporter for [Prometheus](https://prometheus.io). The exporter uses the Redfish API to communicate with iDRAC and it supports the regular `/metrics` endpoint to expose metrics from the host passed via the `target` parameter. For example, to scrape metrics for an iDRAC instance on the IP address `123.45.6.78` call the following URL addresse.
```
http://localhost:9348/metrics?target=123.45.6.78
```

Every time the exporter is called with a new target, it tries to establish a connection to iDRAC. If the target is unreachable or if the authentication fails, the target will be flagged as invalid, and any subsequent call to that target will simply be ignored and a status code 500 is returned.

## Download
The exporter is written in [Go](https://golang.org) and it can be downloaded and compiled using:
```bash
go get github.com/mrlhansen/idrac_exporter
```

## Configuration
In the configuration file for the iDRAC exporter you can specify the bind address and port for the metrics exporter, as well as username and password for all iDRAC hosts. By default the exporter looks for the configuration file in `/etc/prometheus/idrac.yml` but the path can be specified using the `-config` option.
```yaml
address: 127.0.0.1
port: 9348
hosts:
  123.45.6.78:
    username: user
    password: pass
  default:
    username: user
    password: pass
metrics:
  - system
  - sensors
  - sel
```

## List of Metrics
At the moment the exporter only exposes a very limited set of information.

### System
These metrics include power, health, and LED state, total memory size in MiB, number of physical processors and the BIOS version.
```
idrac_power_on 1
idrac_health_ok{status="OK"} 1
idrac_indicator_led_on 0
idrac_memory_size 393216
idrac_cpu_count{model="Intel(R) Xeon(R) Gold 6130 CPU @ 2.10GHz"} 2
idrac_bios_version{version="2.3.10"} NaN
```

### Sensors
These metrics include temperature (in degrees Celcius) and FAN speeds (in RPM).
```
idrac_sensors_temperature{name="Inlet Temp",id="iDRAC.Embedded.1#InletTemp",enabled="1"} 19.0
idrac_sensors_tachometer{name="FAN1A",id="0x17__Fan.Embedded.1A",enabled="1"} 7224
```

### System Event Log
The system event log is also exported. This is not exactly an ordinary metric, but it is often convenient to be informed about new entries in the event log. The value of this metric is the unix timestamp for when the entry was created (as reported by iDRAC).
```
idrac_sel_entry{id="1",message="The process of installing an operating system or hypervisor is successfully completed",component="BaseOSBoot/InstallationStatus",severity="OK"} 1631175352
```
