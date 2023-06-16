#!/bin/bash

config=/etc/prometheus/idrac.yml

if [ ! -e "$config" ]; then 
	auth_file=/authconfig/$NODE_NAME

	if [ ! -e "$auth_file" ]; then
		>&2 echo "$config not found _and_ $auth_file not found."

	else
		# auth_file contents are in the format: user=password
		export IDRAC_USERNAME=$(cut -f1  -d= $auth_file)
		export IDRAC_PASSWORD=$(cut -f2- -d= $auth_file)
		envsubst <${config}.template >${config}
	fi

fi

exec bin/idrac_exporter
