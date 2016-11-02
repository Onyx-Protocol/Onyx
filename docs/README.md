# Documentation

## Development

To view docs with their associated HTML, styles and fonts, we use a tool
called `md2html`.

Make sure all Chain Core commands have been installed by following the
installation instructions in the [repo README](../Readme.md#installation).

Once installed, run `md2html` from the root directory of the rep:

```sh
$ cd $CHAIN
$ go install ./cmd/md2html && md2html
```

The converted documentation is served at
[http://localhost:8080/docs](http://localhost:8080/docs).

## Deployment

### Chain Core

Documentation is bundled into Chain Core inside the `$CHAIN/generated` folder.
To bundle the latest docs, run:

```sh
$ cd $CHAIN
$ ./bin/bundle-docs
```

### Web

#### Dependencies

* [AWS CLI](https://aws.amazon.com/cli/)
* AWS credentials with access to the appropriate buckets

To upload the latest docs to S3, log in to `aws` with the command:

```sh
$ aws configure
```

Once configured, you can upload the docs to the staging bucket with:

```sh
$ cd $CHAIN
$ ./bin/upload-docs
```

To upload to the production bucket instead, run `upload-docs` with `prod` as
an argument:

```sh
$ ./bin/upload-docs prod
```
