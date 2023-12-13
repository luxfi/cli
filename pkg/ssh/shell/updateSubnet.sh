#!/usr/bin/env bash
#name:TASK [stop node - stop node]
sudo systemctl stop node
#name:TASK [import subnet]
/home/ubuntu/bin/lux subnet import file {{ .SubnetExportFileName }} --force
#name:TASK [lux join subnet]
/home/ubuntu/bin/lux subnet join {{ .SubnetName }} --fuji --node-config /home/ubuntu/.node/configs/node.json --plugin-dir /home/ubuntu/.node/plugins --force-write
#name:TASK [restart node - start node]
sudo systemctl start node
