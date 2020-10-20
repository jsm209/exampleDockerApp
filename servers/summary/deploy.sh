sh ./build.sh
docker push jsm209/summary

ssh -tt ec2-user@ec2-3-18-113-171.us-east-2.compute.amazonaws.com << EOF

docker network create myNet

docker rm -f summary
docker pull jsm209/summary
docker run \
    -d \
    -e NAME=summary \
    -e ADDR=:6000 \
    -p 6000:6000 \
    --network myNet \
    --name summary jsm209/summary
EOF