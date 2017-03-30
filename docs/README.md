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

### Web

#### Prerequisites

Prepare the following:

* Install [AWS CLI](https://aws.amazon.com/cli/)
* Have AWS credentials with access to the appropriate buckets

Log into `aws` with the command:

```sh
$ aws configure
```

#### Checking out the right version of docs

Before uploading documentation, make sure your local state reflects the correct documentation. The `main` branch is generally not safe for this purpose, since it may contain documentation updates that reflect changes that have yet to make it into an official release.

The state of production documentation is tracked in the `<major>.<minor>-stable` family of release branches. Each such branch reflects the last known safe version of the documentation for the corresponding major/minor version pair.

For the time being, only the most recent version of the documentation is published online. Please make sure you are on the most recent release branch before uploading.

#### Uploading the docs

Staging:

```
cd $CHAIN
git checkout <major>.<minor>-stable
./bin/upload-docs <major>.<minor>
```

Production:

```
cd $CHAIN
git checkout <major>.<minor>-stable
./bin/upload-docs <major>.<minor> prod
```
