#Testbot
##Introduction
This container runs testbot

##Quickstart
###Build Image
```
$ TAG=chaindev/testbot:`date +%Y%m%d`
$ docker build --tag $TAG $CHAIN/docker/testbot
$ docker push $TAG
$ docker tag $TAG chaindev/testbot:latest
$ docker push chaindev/testbot:latest
```

###Run the container
The container expects several env variables to be set.
Use flag -e to provide them.
```
$ docker pull chaindev/testbot
$ docker run --rm -p 8080:8080 --name testbot\
    -e GITHUB_TOKEN\
    -e SLACK_WEBHOOK_URL\
    -e AWS_ACCESS_KEY_ID\
    -e AWS_SECRET_ACCESS_KEY\
    -e SSH_PRIVATE_KEY\
    chaindev/testbot
```
