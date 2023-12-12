#!/usr/bin/env bash
#name:TASK [stop node]
sudo systemctl stop luxgo
#name:TASK [remove previous luxgo db and logs]
rm -rf /home/ubuntu/.luxgo/db/
rm -rf /home/ubuntu/.luxgo/logs/
#name:TASK [start node]
sudo systemctl start luxgo
