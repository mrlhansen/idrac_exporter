#!/bin/bash

auth_file="/authconfig/$NODE_NAME"
if [ -f "$auth_file" ]; then
	export CONFIG_DEFAULT_USERNAME=$(cut -f1  -d= $auth_file)
	export CONFIG_DEFAULT_PASSWORD=$(cut -f2- -d= $auth_file)
fi

exec bin/idrac_exporter "$@"
