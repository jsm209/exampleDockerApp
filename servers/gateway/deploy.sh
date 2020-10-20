sh ./build.sh
docker push jsm209/gateway
ssh -tt ec2-user@ec2-3-18-113-171.us-east-2.compute.amazonaws.com << EOF
docker rm -f gateway
docker rm -f mysqldemo
docker rm -f redisServer
docker rm -f rabbitmq

docker pull jsm209/gateway

docker network create myNet

docker run -d \
    --network myNet \
    -p 3306:3306 \
    --name mysqldemo \
    -e MYSQL_ROOT_PASSWORD=password \
    -e MYSQL_DATABASE=users \
    jsm209/mysqldemo
sleep 10

docker run -d --name redisServer --network myNet redis
sleep 10

docker run -d \
    --hostname myrabbitmq \
    --name rabbitmq \
    -p 5672:5672 -p 15672:15672 \
    --network myNet\
    rabbitmq:3-management

sleep 10

docker run \
    -d \
    --network myNet \
    -e ADDR=:443 \
    -v /etc/letsencrypt:/etc/letsencrypt:ro \
    -e TLSKEY=/etc/letsencrypt/live/api.infoclass.me/privkey.pem \
    -e TLSCERT=/etc/letsencrypt/live/api.infoclass.me/fullchain.pem \
    -e MESSAGEADDR="http://localhost:5000" \
    -e SUMMARYADDR="http://summary:6000" \
    -e DSN="api.infoclass.me" \
    -e SESSIONKEY="testkey" \
    -e REDISADDR="redisServer:6379" \
    -p 443:443 \
    --name gateway jsm209/gateway
EOF
    