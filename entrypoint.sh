#!/bin/bash

config=/etc/prometheus/idrac.yml

if [[ ! -z "$IDRAC_USERNAME" ]] && [[ ! -z "$IDRAC_PASSWORD" ]]; then
	envsubst <${config}.template > /app/config/idrac.yml
	config=/app/config/idrac.yml
elif [ ! -e "$config" ]; then
	auth_file=/authconfig/$NODE_NAME

	if [ ! -e "$auth_file" ]; then
		>&2 echo "$config not found _and_ $auth_file not found."
	else
		# auth_file contents are in the format: user=password
		export IDRAC_USERNAME=$(cut -f1  -d= $auth_file)
		export IDRAC_PASSWORD=$(cut -f2- -d= $auth_file)
		envsubst <${config}.template > /app/config/idrac.yml
	fi

fi

exec bin/idrac_exporter -config="$config"
