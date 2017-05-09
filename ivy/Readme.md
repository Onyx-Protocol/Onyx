## Ivy

### Run playground:
```
$ brew install yarn
$ yarn install
$ yarn start
$ open http://localhost:8080/
```

### Bundle playround into cored:
```
$ yarn build
$ go install chain/cmd/gobundle
$ gobundle -package ivy playground/public/ > ../generated/ivy/ivy.go
$ gofmt -w ../generated/ivy/ivy.go
```
