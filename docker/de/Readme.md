# Chain Core Developer Edition

## Introduction

Chain Core DE is a docker container that runs Chain Core, exposed on port 1999.

## Quickstart

#### Build the image

```sh
$ sh bin/build-docker-de
```

#### Run the container

>**Note:** The `--name` flag allows you to name the container and refer to that name in subsequent commands.

```sh
$ docker run -p 1999:1999 --name chaincore chaincore/developer
```

Chain Core stores data in three locations: a data directory, a separate Postgres database, and a log directory. To persist the data, create directories on your development machine and mount them to the container on `docker run`:

```sh
$ mkdir -p /path/to/store/datadir
$ mkdir -p /path/to/store/postgres
$ mkdir -p path/to/store/logs
$ docker run -p 1999:1999 \
    -v /path/to/store/datadir:/root/.chaincore \
    -v /path/to/store/postgres:/var/lib/postgresql/data \
    -v /path/to/store/logs:/var/log/chain \
    --name chaincore \
    chaincore/developer
```

#### Access core

A client access token will be printed to your shell. The core and dashboard are listening on `http://localhost:1999`.

#### Stop the container

```sh
$ docker stop chaincore
```

## Other features

#### Save an image

A built image can be saved as a tarball. This allows us to share images between docker clients. To save run:

```sh
$ docker save chaincore/developer > /path/to/file
```

#### Load an image

To load saved images run:

```sh
$ docker load < /path/to/file
```

#### Tail the logs

The container keeps logs for the initial startup and the `cored` process. The client access token is also logged to a file. To access run:

```sh
$ docker exec -it chaincore tail /var/log/chain/init.log     # init logs
$ docker exec -it chaincore tail /var/log/chain/cored.log    # cored logs
$ docker exec -it chaincore tail /var/log/chain/client-token # client access token
```

#### Enter a running container

To receive a command prompt inside the container run:

```sh
$ docker exec -it chaincore /bin/sh
```

#### Check the logs of a stopped container

If a container was exited prematurely, you can receive a command prompt from inside it by running:

```sh
$ docker run -it \
    -v /path/to/store/datadir:/root/.chaincore \
    -v /path/to/store/postgres:/var/lib/postgresql/data \
    -v /path/to/store/logs:/var/log/chain \
    --entrypoint /bin/sh \
    chaincore/developer
```

>**Note:** this command creates a new container and should only be used when persisting the container data.

Then check the logs:

```sh
$ tail /var/log/chain/init.log     # init logs
$ tail /var/log/chain/cored.log    # cored logs
```

