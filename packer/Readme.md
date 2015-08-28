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

### Launching instances

##### Basic config

- IAM role: `chain-ec2-instance`
- Security groups: `splunk`
- SSH key pair: **Proceed without a keypair**
- User data: see below

##### User data

As text:

```sh
#!/bin/sh

# Team SSH access: fetch common authorized_keys
/home/ubuntu/refresh-authorized-keys
```
