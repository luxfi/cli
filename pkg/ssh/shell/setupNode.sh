#!/usr/bin/env bash
#name:TASK [update apt data and install dependencies] 
DEBIAN_FRONTEND=noninteractive sudo apt-get -y update
DEBIAN_FRONTEND=noninteractive sudo apt-get -y install wget curl git
#name:TASK [create .cli .luxgo dirs]
mkdir -p .cli .luxgo/staking
#name:TASK [get lux go script]
wget -nd -m https://raw.githubusercontent.com/luxdefi/lux-docs/master/scripts/luxgo-installer.sh
#name:TASK [modify permissions]
chmod 755 luxgo-installer.sh
#name:TASK [call lux go install script]
./luxgo-installer.sh --ip static --rpc private --state-sync on --fuji --version {{ .LuxGoVersion }}
#name:TASK [get lux cli install script]
wget -nd -m https://raw.githubusercontent.com/luxdefi/cli/main/scripts/install.sh
#name:TASK [modify permissions]
chmod 755 install.sh
#name:TASK [run install script]
./install.sh -n
{{if .IsDevNet}}
#name:TASK [stop luxgo in case of devnet]
sudo systemctl stop luxgo
{{end}}
