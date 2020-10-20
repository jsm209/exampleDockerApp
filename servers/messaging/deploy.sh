sh ./build.sh
docker push jsm209/messages

ssh -tt ec2-user@ec2-3-18-113-171.us-east-2.compute.amazonaws.com << EOF
docker network create myNet

docker rm -f mongodb
docker run -d \
    -p 27018:27018 \
    --name mongodb \
    --network myNet \
    mongo

sleep 10

docker rm -f messages
docker pull jsm209/messages
docker run -d \
    -e ADDR=:5000 \
    -p 5000:5000 \
    --network myNet \
    --name messages jsm209/messages
EOF

    