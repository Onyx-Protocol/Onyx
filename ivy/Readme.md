## Ivy

```
git clone https://github.com/chain/chain.git
cd chain
git checkout ivy
cd ivy
```

To build, bundle, and develop with the playground, we recommend using [yarn](https://yarnpkg.com/en/):

```
$ brew install yarn
```

### Install Chain Core with necessary tags

First, make sure you can build Chain Core from source by following the instructions at https://github.com/chain/chain#building-from-source.

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
$ open http://localhost:8081/
```

### Bundle changes to playround into cored:
```
$ yarn run bundle
```