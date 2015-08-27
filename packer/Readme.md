### Assumptions

- assume aws creds are in the usual place
- chain app runs as ubuntu (which has sudo access)
- no quotes in Procfile command name
- `packer build` may fail occasionally on fetching remote dependencies

### Install packer

```
$ brew update
$ brew install packer
```

### Build AMI

```
$ packer build packer.json
```
