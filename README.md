# iDRAC Exporter
This is a (very) simple iDRAC exporter for [Prometheus](https://prometheus.io). The exporter uses the Redfish API to communicate with iDRAC and it supports the regular `/metrics` endpoint to expose metrics from the host passed via the `target` parameter. For example, to scrape metrics for an iDRAC instance on the IP address `123.45.6.78` call the following URL addresse.
```
http://localhost:9348/metrics?target=123.45.6.78
```

Every time the exporter is called with a new target, it tries to establish a connection to iDRAC. If the target is unreachable or if the authentication fails, the target will be flagged as invalid, and any subsequent call to that target will simply be ignored and a status code 500 is returned.

## Supported Systems
The latest version of the program does not only support iDRAC, but several systems, because they all follow the Redfish standard. The exporter has been tested on the following systems.

* HPE iLO 4/5
* Dell iDRAC 9
* Lenovo XClarity

The `system` and `sensors` metrics (see the details below) are fully supported on all these systems, while the `sel` metrics are limited to iDRAC at the moment.

## Download
The exporter is written in [Go](https://golang.org) and it can be downloaded and compiled using:
```bash
go get github.com/mrlhansen/idrac_exporter
```

## Configuration
In the configuration file for the iDRAC exporter you can specify the bind address and port for the metrics exporter, as well as username and password for all iDRAC hosts. By default the exporter looks for the configuration file in `/etc/prometheus/idrac.yml` but the path can be specified using the `-config` option.
```yaml
address: 127.0.0.1 # Listen address
port: 9348         # Listen port
timeout: 10        # HTTP timeout (in seconds) for Redfish API calls
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

As shown in the example above, under `hosts` you can specify login information for individual hosts via their IP address, otherwise the exporter will attempt to use the login information under `default`. Under `metrics` you can select what kind of metrics that should be returned, as described in more detail below.

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
These metrics include temperature and FAN speeds.
```
idrac_sensors_temperature{name="Inlet Temp",units="celsius"} 19
idrac_sensors_tachometer{name="FAN1A",units="rpm"} 7912
```

### System Event Log
The system event log is also exported. This is not exactly an ordinary metric, but it is often convenient to be informed about new entries in the event log. The value of this metric is the unix timestamp for when the entry was created (as reported by iDRAC).
```
idrac_sel_entry{id="1",message="The process of installing an operating system or hypervisor is successfully completed",component="BaseOSBoot/InstallationStatus",severity="OK"} 1631175352
```

## Prometheus Configuration
For the situation where you have a single `idrac_exporter` and multiple iDRACs to query, the following `prometheus.yml` snippet can be used.

```yaml
scrape_configs:
  - job_name: idrac
    static_configs:
      - targets: ['123.45.6.78', '123.45.6.79']
    relabel_configs:
      - source_labels: [__address__]
        target_label: __param_target
      - source_labels: [__param_target]
        target_label: instance
      - target_label: __address__
        replacement: exporter:9348
```

Here `123.45.6.78` and `123.45.6.79` are the iDRACs to query, and `exporter:9348` is the address and port where `idrac_exporter` is running.
