#!/usr/bin/env bash
set -e
#name:TASK [download new EVM release]
busybox wget "{{ .EVMReleaseURL }}" -O "{{ .EVMArchive }}"
#name:TASK [unpack new EVM release]
tar xvf "{{ .EVMArchive}}"
