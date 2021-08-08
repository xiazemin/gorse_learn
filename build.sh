#cp  /Users/xiazemin/go/bin/gorse-master master
#cp  /Users/xiazemin/go/bin/gorse-server server
#cp  /Users/xiazemin/go/bin/gorse-worker worker

cd gorse
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./bin/ ./...
cd ..
cp gorse/bin/gorse-master master
cp gorse/bin/gorse-server server
cp gorse/bin/gorse-worker worker


docker build -t gorse_master -f gorse-master/Dockerfile .
docker build -t gorse_server -f gorse-server/Dockerfile .
docker build -t gorse_worker -f gorse-worker/Dockerfile .
docker-compose up -d

 docker ps
docker exec -it a5adaf891c63 /bin/bash

#ADD ./github.sql /etc/github.sql
RUN mysql -h 127.0.0.1 -u root -e "create database gorse;"

RUN mysql -h 127.0.0.1 -u root gorse < github.sql

docker build -t mysql_gitrec -f mysql.Dockerfile .