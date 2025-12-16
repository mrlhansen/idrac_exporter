#!/bin/sh

auth_file="/authconfig/${NODE_NAME}"
if [ -f "${auth_file}" ]; then
	CONFIG_DEFAULT_USERNAME="$(cut -f1 -d= "${auth_file}")"
	CONFIG_DEFAULT_PASSWORD="$(cut -f2- -d= "${auth_file}")"
	export CONFIG_DEFAULT_USERNAME CONFIG_DEFAULT_PASSWORD
fi

exec bin/idrac_exporter "$@"