# Install Chain Core on RHEL with Systemd

This guide shows how to install Chain Core and how to configure systemd to manage the starting and stopping of the cored process.

1. Download cored to `/usr/bin/`
2. Configure cored using `/etc/cored.env`
3. Configure systemd using `/etc/systemd/system/cored.service`

## Downloading cored

```
$ curl -o /usr/bin/cored -L https://s3.amazonaws.com/chain-core/1.0.2/cored
$ chmod +x /usr/bin/cored
```

## Configuring cored

Here is an example `/etc/cored.env` file:
```
export DATABASE_URL=postgres://{USERNAME}:{PASSWORD}@{HOST}:{PORT}/{DATABASE-NAME}?sslmode=disable
```
The minimal cored configuration requires `DATABASE_URL`.

## Configure systemd

```
[Unit]
Description=Chain Core API Server
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
