#!/usr/bin/env bash
#name:TASK [stop node]
sudo systemctl stop node
#name:TASK [remove previous node db and logs]
rm -rf /home/ubuntu/.node/db/
rm -rf /home/ubuntu/.node/logs/
#name:TASK [start node]
sudo systemctl start node
