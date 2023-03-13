# iDRAC Exporter
This is a simple iDRAC exporter for [Prometheus](https://prometheus.io). The exporter uses the Redfish API to communicate with iDRAC and it supports the regular `/metrics` endpoint to expose metrics from the host passed via the `target` parameter. For example, to scrape metrics for an iDRAC instance on the IP address `123.45.6.78` call the following URL addresse.
```
http://localhost:9348/metrics?target=123.45.6.78
```

Every time the exporter is called with a new target, it tries to establish a connection to iDRAC. If the target is unreachable or if the authentication fails, the target will eventually be flagged as invalid, and any subsequent call to that target will simply be ignored and a status code 500 is returned.

## Supported Systems
The latest version of the program does not only support iDRAC, but several systems, because they all follow the Redfish standard. The exporter has been tested on the following systems.

* HPE iLO 4/5
* Dell iDRAC 9
* Lenovo XClarity

## Download
The exporter is written in [Go](https://golang.org) and it can be downloaded and compiled using:
```bash
go install github.com/mrlhansen/idrac_exporter/cmd/idrac_exporter@latest
```

## Configuration
In the configuration file for the iDRAC exporter you can specify the bind address and port for the metrics exporter, as well as username and password for all iDRAC hosts. By default the exporter looks for the configuration file in `/etc/prometheus/idrac.yml` but the path can be specified using the `-config` option.
```yaml
address: 127.0.0.1 # Listen address
port: 9348         # Listen port
timeout: 10        # HTTP timeout (in seconds) for Redfish API calls
retries: 1         # Number of retries before a target is marked as unreachable
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
  - power
  - sel            # iDRAC only
  - drives
```

As shown in the example above, under `hosts` you can specify login information for individual hosts via their IP address, otherwise the exporter will attempt to use the login information under `default`. Under `metrics` you can select what kind of metrics that should be returned, as described in more detail below.

## List of Metrics
At the moment the exporter exposes the following list of metrics.

### System
These metrics include power, health, and LED state, total memory size, number of physical processors, BIOS version and machine information.
```
idrac_power_on 1
idrac_health_ok{status="OK"} 1
idrac_indicator_led_on{state="Lit"} 1
idrac_memory_size_bytes 137438953472
idrac_cpu_count{model="Intel(R) Xeon(R) Gold 6130 CPU @ 2.10GHz"} 2
idrac_bios_info{version="2.3.10"} 1
idrac_machine_info{manufacturer="Dell Inc.",model="PowerEdge C6420",serial="abc",sku="xyz"} 1
```

### Sensors
These metrics include temperature and FAN speeds.
```
idrac_sensors_temperature{name="Inlet Temp",units="celsius"} 19
idrac_sensors_fan_speed{name="FAN1A",units="rpm"} 7912
```

### Power
These metrics include two sets of power readings. The first set is PSU power readings, such as power usage, total power capacity, input voltage and efficiency. Be aware that not all metrics are available on all systems.
```
idrac_power_supply_output_watts{psu="0"} 74.5
idrac_power_supply_input_watts{psu="0"} 89
idrac_power_supply_capacity_watts{psu="0"} 750
idrac_power_supply_input_voltage{psu="0"} 232
idrac_power_supply_efficiency_percent{psu="0"} 91
```

The second set is the power consumption for the entire system (and sometimes also for certain subsystems, such as the CPUs). The first two metrics are instantaneous readings, while the last four metrics are the minimum, maximum and average power consumption as measure over the reported interval.
```
idrac_power_control_consumed_watts{id="0",name="System Power Control"} 166
idrac_power_control_capacity_watts{id="0",name="System Power Control"} 816
idrac_power_control_min_consumed_watts{id="0",name="System Power Control"} 165
idrac_power_control_max_consumed_watts{id="0",name="System Power Control"} 177
idrac_power_control_avg_consumed_watts{id="0",name="System Power Control"} 166
idrac_power_control_interval_in_minutes{id="0",name="System Power Control"} 1
```

### System Event Log
On iDRAC only, the system event log can also be exported. This is not exactly an ordinary metric, but it is often convenient to be informed about new entries in the event log. The value of this metric is the unix timestamp for when the entry was created (as reported by iDRAC).
```
idrac_sel_entry{id="1",message="The process of installing an operating system or hypervisor is successfully completed",component="BaseOSBoot/InstallationStatus",severity="OK"} 1631175352
```

### Drive status
These metrics include information about disk drives in the machine. The value represents health: 0 is OK, 1 is Warning, 2 is Critical, 10 is unknown status.
```
idrac_drive_health{capacity="299439751168",health="Critical",manufacturer="TOSHIBA",mediatype="HDD",model="AL14SXB30ENY",name="Physical Disk 0:1:24",slot="24",state="StandbyOffline"} 2
idrac_drive_health{capacity="299439751168",health="OK",manufacturer="TOSHIBA",mediatype="HDD",model="AL14SXB30ENY",name="Physical Disk 0:1:25",slot="25",state="Enabled"} 0
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
