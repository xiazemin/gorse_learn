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

docker pull redis
docker pull mysql:5.7
#docker pull mysql/mysql-server:5.7

docker-compose up -d

docker ps
docker exec -it 82c5660e31d5 /bin/bash

docker cp ./gitrec/init/github.sql  82c5660e31d5:/var

mysql -h 127.0.0.1 -u gorse -pgorse_pass gorse < /var/github.sql

cd gitrec
docker build .

#ADD ./github.sql /etc/github.sql
RUN mysql -h 127.0.0.1 -u root -e "create database gorse;"

RUN mysql -h 127.0.0.1 -u root gorse < github.sql

docker build -t mysql_gitrec -f mysql.Dockerfile .

cd frontend
npm install
npm install vue-cli
npm run build
npm run serve

http://localhost:8080/


SECRET_KEY=SECRET_KEY
GITHUB_OAUTH_CLIENT_SECRET=GITHUB_OAUTH_CLIENT_SECRET
GITHUB_OAUTH_CLIENT_ID=GITHUB_OAUTH_CLIENT_ID
GITHUB_ACCESS_TOKEN=GITHUB_ACCESS_TOKEN

pip3 install gunicorn
pip3 install gevent
pip3 install -r requirements.txt
PYTHONPATH=backend gunicorn backend.app:app -c gunicorn.conf.py

http://0.0.0.0:5000/login
