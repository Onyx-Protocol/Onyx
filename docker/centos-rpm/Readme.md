#Chain Core Developer Edition - CentOS RPM builder
##Introduction
This image is used to build an rpm compatible with CentOS versions 5.11 and 6.8
The resulting rpm includes:
- the `cored` binary
- the `corectl` binary
- a SysVinit script

A configuration file `/etc/cored.env` must be added to the intended
host. It should contain relevant export commands for env setup.

For example:
```
export DATABASE_URL=...
export LISTEN=:<port>
```

##Build image
```
$ docker build --tag centos-rpm $CHAIN/docker/centos-rpm/
```

##Build rpm
Upon running the container, a host directory must be mounted to the
container's `/output` directory. The rpm will be generated there. The
rpm will also need a version number passed as the VERSION env var.
```
$ docker run -it --rm -v /path/to/output/dir:/output -e VERSION=XXXXXXXX centos-rpm
```
