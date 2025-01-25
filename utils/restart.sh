#!/usr/bin/env bash

sudo pm2 stop video-service
sudo GOMAXPROCS=3 pm2 start video-service-linux-amd64 --name=video-service -- -config=./prod.ini
sudo pm2 save