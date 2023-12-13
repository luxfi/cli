#!/usr/bin/env bash
#name:TASK [update apt data and install dependencies] 
DEBIAN_FRONTEND=noninteractive sudo apt-get -y update
DEBIAN_FRONTEND=noninteractive sudo apt-get -y install wget curl git
#name:TASK [create .cli .node dirs]
mkdir -p .cli .node/staking
#name:TASK [get lux go script]
wget -nd -m https://raw.githubusercontent.com/luxdefi/lux-docs/master/scripts/node-installer.sh
#name:TASK [modify permissions]
chmod 755 node-installer.sh
#name:TASK [call lux go install script]
./node-installer.sh --ip static --rpc private --state-sync on --fuji --version {{ .LuxdVersion }}
#name:TASK [get lux cli install script]
wget -nd -m https://raw.githubusercontent.com/luxdefi/cli/main/scripts/install.sh
#name:TASK [modify permissions]
chmod 755 install.sh
#name:TASK [run install script]
./install.sh -n
{{if .IsDevNet}}
#name:TASK [stop node in case of devnet]
sudo systemctl stop node
{{end}}
