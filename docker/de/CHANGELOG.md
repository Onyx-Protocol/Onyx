# Chain Core Developer Edition - Docker Image changelog

## 1.2.1 (June 13, 2017)

* Update to Chain Core [1.2.1](https://chain.com/docs/1.2/core/reference/changelog#1.2.1)

## 1.2.0 (May 12, 2017)

* Update to Chain Core [1.2.0](https://chain.com/docs/1.2/core/reference/changelog#1.2.0)

#### Local filesystem requirement

Chain Core now uses an additional `datadir` for storing non-postgres data. **Docker users** should take care to mount a volume for the Chain Core data directory, or else the server state will not persist between runs of the container:

```
$ docker run --rm -p 1999:1999 \
    -v /path/to/store/datadir:/root/.chaincore \
    -v /path/to/store/postgres:/var/lib/postgresql/data \
    -v /path/to/store/logs:/var/log/chain \
    --name chaincore \
    chaincore/developer
```
