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

#### The `docs-release` branch

Before uploading documentation, make sure your local state reflects the correct documentation. The `main` branch is generally not safe for this purpose, since it may contain documentation updates that reflect changes that have yet to make it into an official relase.

The state of production documentation is tracked in the `docs-release` branch. This branch that reflects the last known safe version of the documentation. Typically, it will contain the contents of `main`, **minus** updates to the `docs` and `sdk` directories that reflect unreleased updates.

Since this contents of `docs-release` are assembled ad hoc, the history of this branch is relatively unimportant. It's fine to use force-pushes to synchronize the branch with `main`. Our current convention is to find a stable baseline commit on `main`, and then add cherry-picked commits that refer to commits in `main` that contain releasable updates to the `docs` and `sdk` directories.

#### Uploading the docs

Staging:

```
cd $CHAIN
git checkout docs-release
./bin/upload-docs
```

Production:

```
cd $CHAIN
git checkout docs-release
./bin/upload-docs prod
```
