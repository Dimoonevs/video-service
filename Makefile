HOST=ec2-13-49-78-32.eu-north-1.compute.amazonaws.com
HOMEDIR=/var/www/video-service/
USER=dima

video-service-linux:
	GOOS=linux GOARCH=amd64 go build -o bin/video-service-linux-amd64 ./

upload-video-service: video-service-linux
	rsync -rzv --progress --rsync-path="sudo rsync" \
		./bin/video-service-linux-amd64  \
		./utils/cfg/prod.ini \
		./utils/restart.sh \
		$(USER)@$(HOST):$(HOMEDIR)

restart-video-service:
	echo "sudo su && cd $(HOMEDIR) && bash restart.sh && exit" | ssh $(USER)@$(HOST) /bin/sh

upload-and-restart: upload-video-service restart-video-service

run-local:
	go run main.go -config ./utils/cfg/local.ini