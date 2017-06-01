This Docker image is used to generate the chain.com/docs website,
and the documentation for individual SDKs.

Build this WIP image with:

```
docker build -t docs-wip .
```

Run the image with an output directory mapped to `/generated` like below:

```
docker run -v /absolute/path/to/output:/generated docs-wip
```
