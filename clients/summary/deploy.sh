sh ./build.sh
docker push jsm209/summary
ssh -tt ec2-user@ec2-3-134-109-165.us-east-2.compute.amazonaws.com << EOF
docker rm -f summary
docker pull jsm209/summary
docker run \
    -d \
    -p 443:443 \
    -p 80:80 \
    -v /etc/letsencrypt:/etc/letsencrypt:ro \
    --name summary jsm209/summary
EOF