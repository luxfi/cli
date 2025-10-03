#!/usr/bin/env bash

# Provide docker-compose systemctl unit file
cat << EOF | sudo tee /etc/systemd/system/lux-cli-docker.service
[Unit]
Description=Lux CLI Docker Compose Service
Requires=docker.service
After=docker.service

[Service]
User=ubuntu
Group=ubuntu
Restart=on-failure
ExecStart=/usr/bin/docker compose -f /home/ubuntu/.lux-cli/services/docker-compose.yml up 
ExecStop=/usr/bin/docker compose -f /home/ubuntu/.lux-cli/services/docker-compose.yml down

[Install]
WantedBy=default.target
EOF

sudo systemctl enable lux-cli-docker.service
