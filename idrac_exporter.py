#!/usr/bin/env python3

from http.server import HTTPServer, BaseHTTPRequestHandler
import yaml
import base64
import re
import requests
import urllib3
import dateutil.parser
import time
import argparse


################################################################################


class RedfishAPI:
    def __init__(self, server):
        self.server = server
        self.baseurl = f'https://{server}/redfish/v1'
        self.sslcert = False
        self.session = requests.Session()

    def authenticate(self, token):
        self.token = token
        try:
            result = self.get('Chassis')
        except:
            return False
        if not result.get('error'):
            return True
        self.token = None
        return False

    def get(self, url):
        url = f'{self.baseurl}/{url}'
        header = { 'Authorization' : f'Basic {self.token}', 'Accept' : 'application/json' }
        result = self.session.get(url, headers=header, verify=self.sslcert, timeout=1)
        code = result.status_code
        if code == 200:
            code = None
        retval = {}
        retval['json'] = result.json()
        retval['error'] = code
        return retval

    def metrics_clear(self):
        self.metrics = []

    def metrics_get(self):
        return "\n".join(self.metrics)+"\n"

    def metrics_add(self, name, args, value):
        if args:
            args = ','.join('%s="%s"' % (k,v) for k,v in args.items())
            args = '{%s}' % args
        else:
            args = ''
        if value is None:
            value = 'NaN'
        self.metrics.append('idrac_%s%s %s' % (name, args, value))

    def collect_sel(self):
        data = self.get('Managers/iDRAC.Embedded.1/Logs/Sel')
        error = data.get('error')
        data = data.get('json')
        if error:
            return

        data = data.get('Members')
        for entry in data:
            args = { 'id' : entry['Id'], 'message' : entry['Message'].strip('.'), 'component' : entry['SensorType'], 'severity' : entry['Severity'] }
            value = dateutil.parser.isoparse(entry['Created']).timetuple()
            value = int(time.time()-time.mktime(value))
            self.metrics_add('sel_entry', args, value)

    def collect_system(self):
        data = self.get('Systems/System.Embedded.1')
        error = data.get('error')
        data = data.get('json')
        if error:
            return

        value = 1 if data['PowerState'] == 'On' else 0
        self.metrics_add('power_on', {}, value)

        text = data['Status']['Health']
        value = 1 if text == 'OK' else 0
        self.metrics_add('health_ok', { 'status' : text }, value)

        value = 0 if data['IndicatorLED'] == 'Off' else 1
        self.metrics_add('indicator_led_on', {}, value)

        text = data['MemorySummary']['TotalSystemMemoryGiB']
        value = int(float(text)*1024**4/10**9)
        self.metrics_add('memory_size', {}, value)

        text = data['ProcessorSummary']['Model']
        value = data['ProcessorSummary']['Count']
        self.metrics_add('cpu_count', { 'model' : text }, value)

        text = data['BiosVersion']
        self.metrics_add('bios_version', { 'version' : text }, None)

    def collect_sensors(self):
        data = self.get('Dell/Systems/System.Embedded.1/DellNumericSensorCollection')
        error = data.get('error')
        data = data.get('json')
        if error:
            return
        data = data.get('Members')
        for entry in data:
            args = { 'name' : entry['ElementName'], 'id' : entry['DeviceID'] }
            args['enabled'] = 1 if entry['EnabledState'] == 'Enabled' else 0
            if entry['SensorType'] == 'Temperature':
                value = float(entry['CurrentReading'])/10.0
                name = 'sensors_temperature'
            elif entry['SensorType'] == 'Tachometer':
                value = int(entry['CurrentReading'])
                name = 'sensors_tachometer'
            else:
                continue
            if value < 0:
                value = 0
            self.metrics_add(name, args, value)


################################################################################


def encode_token(username, password):
    token = f'{username}:{password}'
    token = token.encode('ascii')
    token = base64.b64encode(token)
    token = token.decode('ascii')
    return token

def collect_metrics(host):
    metrics = config['metrics']
    hosts = config['hosts']
    retval = ""
    if not hosts.get(host):
        hosts[host] = hosts['default'].copy()
    info = hosts.get(host)
    if 'host' not in info:
        host = RedfishAPI(host)
        valid = True
        if not host.authenticate(info['token']):
            valid = False
        info['host'] = host
        info['valid'] = valid
    if info.get('valid'):
        host = info['host']
        host.metrics_clear()
        if 'system' in metrics:
            host.collect_system()
        if 'sensor' in metrics:
            host.collect_sensors()
        if 'sel' in metrics:
            host.collect_sel()
        retval = host.metrics_get()
    else:
        raise RuntimeError('Invalid host')
    return retval


################################################################################


class ServiceHandler(BaseHTTPRequestHandler):
    def do_GET(self):
        metrics = ""
        p = re.match('/([a-z]+)(\?target=(.*))?', self.path)
        b = p.group(1)
        p = p.group(3)
        if b == 'metrics':
            if not p:
                status = 400
            else:
                status = 200
        else:
            status = 404
        if status == 200:
            try:
                metrics = collect_metrics(p)
            except:
                status = 500
        self.send_response(status)
        self.send_header('Content-type', 'text/plain')
        self.end_headers()
        self.wfile.write(metrics.encode())

    def log_message(self, format, *args):
        return


################################################################################


parser = argparse.ArgumentParser(description='iDRAC exporter for Prometheus monitoring system')
parser.add_argument('--config', help='path to idrac exporter configuration file')
args = parser.parse_args()

if args.config:
    configfile = args.config
else:
    configfile = '/etc/prometheus/idrac.yml'

try:
    file = open(configfile, 'r')
except:
    print(f'{configfile}: unable to open configuration file')
    exit(1)

try:
    config = yaml.full_load(file)
    file.close()
except:
    print(f'{configfile}: unable to parse configuration file')
    exit(1)

if not config.get('address'):
    config['address'] = '0.0.0.0'

if not config.get('port'):
    config['port'] = 9348

if not config.get('hosts'):
    print(f'{configfile}: missing section: hosts')
    exit(1)

if not config.get('metrics'):
    print(f'{configfile}: missing section: metrics')
    exit(1)

for key,value in config['hosts'].items():
    if 'username' not in value or 'password' not in value:
        print(f'{configfile}: username or password missing for host: {key}')
        exit(1)
    value['token'] = encode_token(value['username'], value['password'])

for key in config['metrics']:
    if key not in ('system', 'sensor', 'sel'):
        print(f'{configfile}: unknown metrics component: {key}')
        exit(1)

try:
    urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)
    server = HTTPServer((config['address'], config['port']), ServiceHandler)
    server.serve_forever()
except Exception as e:
    print(f'Unable to start HTTP server: {e}')
    exit(3)
