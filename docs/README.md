# Documentation

## Development

To view docs with their associated HTML, styles and fonts, we use a tool called `md2html`.

Make sure all Chain Core commands have been installed by following the installation instructions in the [repo README](../Readme.md#installation).

Once installed, run `md2html` from the root directory of the rep:

```sh
$ cd $CHAIN
$ go install ./cmd/md2html && md2html
```

The converted documentation is served at [http://localhost:8080/docs](http://localhost:8080/docs).

## Deployment

### Chain Core

Documentation is bundled into Chain Core inside the `$CHAIN/generated` folder. To bundle the latest docs, run:

```sh
$ cd $CHAIN
$ ./bin/bundle-docs
```

### Web

#### Prerequisites

Prepare the following:

* Install [AWS CLI](https://aws.amazon.com/cli/)
* Have AWS credentials with access to the appropriate buckets

Log into `aws` with the command:

```sh
$ aws configure
```

#### The `docs-<major>.<minor>.x` branches

Before uploading documentation, make sure your local state reflects the correct documentation. The `main` branch is generally not safe for this purpose, since it may contain documentation updates that reflect changes that have yet to make it into an official relase.

The state of production documentation is tracked in the `docs-<major>.<minor>.x` family of branches. Each such branch reflects the last known safe version of the documentation for the corresponding major/minor version pair.

#### Uploading the docs

Staging:

```
cd $CHAIN
git checkout docs-<major>.<minor>.x
./bin/upload-docs
```

Production:

```
cd $CHAIN
git checkout docs-<major>.<minor>.x
./bin/upload-docs prod
```
