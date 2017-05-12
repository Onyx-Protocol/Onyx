## Ivy

To build, bundle, and develop with the playground, we recommend using [yarn](https://yarnpkg.com/en/):

```
$ brew install yarn
```

### Install Chain Core with necessary tags

```
$ go install -tags "localhost_auth http_ok" chain/cmd/cored
```

### Run playground in development mode:

```
$ cored
```

```
$ yarn install
$ yarn start
```

```
$ open http://localhost:8080/
```

### Bundle changes to playround into cored:
```
$ yarn run bundle
```