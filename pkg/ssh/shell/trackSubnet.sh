#!/usr/bin/env bash
export PATH=$PATH:~/go/bin:~/.cargo/bin
/home/ubuntu/bin/lux subnet import file {{ .SubnetExportFileName }} --force
sudo systemctl stop luxgo
/home/ubuntu/bin/lux subnet join {{ .SubnetName }} {{ .NetworkFlag }} --luxgo-config /home/ubuntu/.luxgo/configs/node.json --plugin-dir /home/ubuntu/.luxgo/plugins --force-write
sudo systemctl start luxgo
