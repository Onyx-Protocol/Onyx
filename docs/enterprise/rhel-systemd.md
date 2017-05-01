# Install Chain Core on RHEL with Systemd

This guide shows how to install Chain Core and how to configure systemd to manage the starting and stopping of the cored process.

1. Install Postgres 9.6
2. Download cored & corectl to `/usr/bin/`
3. Configure cored using `/etc/cored.env`
4. Configure systemd using `/etc/systemd/system/cored.service`

## Install Postgres 9.6

```
# curl -LO https://download.postgresql.org/pub/repos/yum/9.6/redhat/rhel-6-x86_64/pgdg-centos96-9.6-3.noarch.rpm
# service postgresql-9.6 initdb
# chkconfig postgresql-9.6 on
# echo host all all 127.0.0.1/32 trust > /var/lib/pgsql/9.5/data/pg_hba.conf
# service postgresql-9.6 on
# psql postgres://postgres:@localhost -c 'create database core'
```

Taken from the official Postgres instructions:
https://wiki.postgresql.org/wiki/YUM_Installation


## Downloading cored

```
$ curl -LO https://download.chain.com/bin/chain-core-server-1.1.3.tar.gz
$ tar -xzf chain-core-server-1.1.3.tar.gz

$ cp chain-core-server-1.1.3/cored /usr/bin/cored
$ chmod +x /usr/bin/cored

$ cp chain-core-server-1.1.3/corectl /usr/bin/corectl
$ chmod +x /usr/bin/corectl
```

## Configuring cored

Here is an example `/etc/cored.env` file:
```
export DATABASE_URL=postgres://{USERNAME}:{PASSWORD}@{HOST}:{PORT}/{DATABASE-NAME}?sslmode=disable
```
The minimal cored configuration requires `DATABASE_URL`.

If you have followed the instructions in the Install Postgres 9.6 section, you can use this line:

```
export DATABASE_URL=postgres://postgres:@localhost/core?sslmode=disable
```

## Configure systemd

```
[Unit]
Description=Chain Core Server
After=network.target

[Service]
User=cored
EnvironmentFile=/etc/cored.env
ExecStart=/usr/bin/cored
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

## Start cored

```
systemctl start cored
```

## View cored logs

```
journalctl -u cored.service
```
