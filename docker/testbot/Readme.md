#Testbot
##Introduction
This container runs testbot

##Quickstart
###Build Image
```
$ docker build --tag testbot $CHAIN/docker/testbot/
```

###Run the container
The container expects the `GITHUB_TOKEN` and `SLACK_WEBHOOK_URL` env variables to be set.
```
$ docker run --rm -p 8080:8080 --name testbot -e GITHUB_TOKEN -e SLACK_WEBHOOK_URL testbot
```
