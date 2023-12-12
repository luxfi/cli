#!/usr/bin/env bash
#name:TASK [stop node - stop luxgo]
sudo systemctl stop luxgo
#name:TASK [import subnet]
/home/ubuntu/bin/lux subnet import file {{ .SubnetExportFileName }} --force
#name:TASK [lux join subnet]
/home/ubuntu/bin/lux subnet join {{ .SubnetName }} --fuji --luxgo-config /home/ubuntu/.luxgo/configs/node.json --plugin-dir /home/ubuntu/.luxgo/plugins --force-write
#name:TASK [restart node - start luxgo]
sudo systemctl start luxgo
