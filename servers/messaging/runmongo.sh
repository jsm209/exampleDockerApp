ssh -tt ec2-user@ec2-3-15-172-80.us-east-2.compute.amazonaws.com << EOF
docker rm -f customMongoContainer
docker network create mongoNet
docker run -d \
    -p 27017:27017 \
    --name customMongoContainer \
    --network mongoNet
    mongo
EOF


