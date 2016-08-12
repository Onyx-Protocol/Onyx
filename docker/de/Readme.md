#Chain Core Developer Edition
##Introduction
Chain Core DE is a docker container that runs `cored`, exposed on port 8080.

##Quickstart
####Build the image
```
$ sh bin/build-ccde
```

####Find the image id
```
$ docker images

REPOSITORY          TAG                 IMAGE ID            CREATED             SIZE
chain-core-de       r186-26-g2692ccf    746e65bf0e53        9 minutes ago       194.7 MB
alpine              latest              f70c828098f5        10 days ago         4.799 MB
```

####Run the container
>**Note:** The `--name` flag allows you to name the container and refer to that name in subsequent commands.

```
$ docker run --rm -p 8080:8080 --name <NAME> <IMAGE_ID>
```

####Access core
The api credentials will be printed to your shell. The core is listening on `http://localhost:8080`

####Stop the container
```
$ docker stop <NAME>
```
>**Note:** By default, once you stop a running a container, all data is lost. To persist the data, create directories on your development machine and mount them to the container on `docker run`

```
$ mkdir -p /path/to/store/db
$ mkdir -p path/to/store/logs
$ docker run --rm -p 8080:8080 -v /path/to/store/db:/var/lib/postgresql/data -v /path/to/store/logs:/var/log/chain --name <NAME> <IMAGE_ID>
```

##Other features
####Save an image
A built image can be saved as a tarball. This allows us to share images between docker clients. To save run:
```
$ docker save <IMAGE_ID> > /path/to/file
```

####Load an image
To load saved images run:
```
$ docker load < /path/to/file
```

####Tail the logs
The container keeps logs for both processes and a copy of the core credentials. To access run:
```
$ docker exec -it <NAME> tail /var/log/chain/cored.log # cored logs
$ docker exec -it <NAME> tail /var/log/chain/credentials.json # core credentials
```

####Enter a container
To receive a command prompt inside the container run:
```
$ docker exec -it <NAME> /bin/sh
```
