# Connect Chain Core to an external PostgreSQL server
This guide shows how to run Chain Core on 2 servers:

1. A server running the Chain Core binary
2. A server running PostgreSQL

For this example, we will assume that server #1 runs on `10.0.0.1` and server #2 runs on `10.0.0.2`.

### Setting up the PostgreSQL server
The PostgreSQL server needs to be running PostgreSQL 9.5 and it must expose port `5432` to the server running the Chain Core binary. Here is Centos/RHEL example:

```
sudo yum -y install http://yum.postgresql.org/9.5/redhat/rhel-6.8-x86_64/pgdg-redhat95-9.5-2.noarch.rpm
sudo yum -y install postgresql95-server postgresql95-contrib
```
Once PostgreSQL is installed, we can initialize the PostgreSQL database and allow incoming connection from our Chain Core server.
```
sudo service postgresql-9.5 initdb
sudo su -c 'echo host all all 10.0.0.1/32 trust > /var/lib/pgsql/9.5/data/pg_hba.conf'
sudo su -c 'echo host all all 127.0.0.1/32 trust > /var/lib/pgsql/9.5/data/pg_hba.conf'
sudo service postgresql-9.5 on
psql postgres://postgres:@127.0.0.1 -c 'create database core'
```

### Setting up the Chain Core server
Now that we have a server running on `10.0.0.2:5432`, we can instruct the Docker container that is running Chain Core to connect to the PostgreSQL server.

```
$ docker run \
  -it \
  -p 1999:1999 \
  -e DATABASE_URL=postgres://10.0.0.2:5432/core \
  chaincore/developer
```
