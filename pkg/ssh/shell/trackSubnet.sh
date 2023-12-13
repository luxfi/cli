#!/usr/bin/env bash
export PATH=$PATH:~/go/bin:~/.cargo/bin
/home/ubuntu/bin/lux subnet import file {{ .SubnetExportFileName }} --force
sudo systemctl stop node
/home/ubuntu/bin/lux subnet join {{ .SubnetName }} {{ .NetworkFlag }} --node-config /home/ubuntu/.node/configs/node.json --plugin-dir /home/ubuntu/.node/plugins --force-write
sudo systemctl start node
