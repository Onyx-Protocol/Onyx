# Operations

* [Chain Core Testnet](#chain-core-testnet)
  * [Network Info](#network-info)
  * [Reset](#reset)
  * [Updating the Generator](#updating-the-generator)
  * [Updating the Signers](#updating-the-signers)
* [Creating Networks](#creating-networks)
  * [Unauthenticated](#unauthenticated)
  * [Authenticated](#authenticated)
* [Building a new release](#building-a-new-release)

## Chain Core Testnet
### Network Info
The relevant information for joining the testnet can be
found at `https://testnet-info.chain.com`.
This includes the generator's url and network access
token and the network's blockchain id.

Information regarding the generator and signers, e.g.
access tokens & block signing pubkeys,
can be located within the config vars of our reset tool:
```
$ heroku config -a testnet-resetter
```
>**Note:** Testnet is reset every week. The access
>tokens and signing keys for the
key participants do not change, but the blockchain id
will need to be updated in the
testnet info config variables.

### Reset
To reset the network we use the cmd `testnet-reset`. The binary is deployed to Heroku.

To reset the network run:
```
$ heroku run bin/testnet-reset -a testnet-resetter
```

Next, the blockchain id will need to be updated in the testnet-info app.

To retrieve the blockchain id run:
```
$ BLOCKCHAIN_ID=`curl --silent --user $GENERATOR_CLIENT_TOKEN $GENERATOR_URL/info | jq -r .blockchain_id`
```

To update the blockchain id:
```
$ heroku config:set BLOCKCHAIN_ID=$BLOCKCHAIN_ID -a chain-testnet-info
```

### Updating the Generator
The generator's code is deployed through the `deploy`
command. `deploy`
builds the `cored` binary from the caller's local
environment and uses
ssh to copy the binary to the address specified. This
guide will walk
you through the deployment steps.

#### Set configuration variables
```
$ export CHAIN=path/to/chain/src
$ export INSTANCE_ADDR=...
```
>Note: In addition to the above env vars, the command
>needs to authenticate
its connection to the generator's host. Be sure to set
your ssh private key
as an environment variable (`SSH_PRIVATE_KEY`) or have
ssh-agent configured
on your machine. Use the same key authorized for other
Chain ec2 instances.

#### Checkout the latest code.

#### Install command
```
$ cd $CHAIN
$ go install ./cmd/deploy
```

#### Checkout code to deploy.

#### Run command:
```
$ deploy
```

### Updating the Signers
The signers for our testnet run our Chain Core DE docker
image. A new version
of the image must be built and uploaded to s3.

#### Build the image
```
$ cd $CHAIN
$ bin/build-ccde
```

#### Export the image
```
$ docker save chain:latest -o path/to/dest/latest.tar
```

#### Upload the image to s3
```
$ aws s3 cp path/to/latest.tar s3://chain-core/YYYYMMDD/latest.tar --acl public-read
```

Once the image has been uploaded to s3, each signer must
follow the process
below to update their core.

#### Download latest Chain Core Docker image
```
$ curl -LO https://s3.amazonaws.com/chain-core/YYYYMMDD/latest.tar
```

#### Delete current container
```
$ docker stop chain
$ docker rm chain
```

#### Load new image into Docker engine
```
$ docker load < latest.tar
```

#### Start new container
```
$ docker run -d -p 1999:1999 \
    -v /var/lib/chain/postgresql/data:/var/lib/postgresql/data \
    -v /var/log/chain/:/var/log/chain \
    --name chain \
    --restart always \
    chain:latest
```

## Creating Networks
### Unauthenticated
This guide will demonstrate creating a 2-of-3 block
signing network with a generator and two signers.
We assume you have database access to 3 running,
unconfigured cores. We will also makes use of the
`corectl`, `curl`
and `jq` commands. Please note that Chain Core only
allows unauthenticated connections on the loopback
interface. If you
will be connecting to the cores through another network
device, please follow the instructions
[here](#authenticated).

#### Define configuration variables
```
$ GENERATOR_URL=...
$ GENERATOR_DB_URL=...
$ SIGNER1_URL=...
$ SIGNER1_DB_URL=...
$ SIGNER2_URL=...
$ SIGNER2_DB_URL=...
```

#### Create block signing keypairs
Generator:
```
$ GENERATOR_PUBKEY=`DATABASE_URL=$GENERATOR_DB_URL corectl create-block-keypair`
```

Signer 1:
```
$ SIGNER1_PUBKEY=`DATABASE_URL=$SIGNER1_DB_URL corectl create-block-keypair`
```

Signer 2:
```
$ SIGNER2_PUBKEY=`DATABASE_URL=$SIGNER2_DB_URL corectl create-block-keypair`
```

#### Configure the generator
```
$ curl --silent $GENERATOR_URL/configure --data '{
    "is_generator":true,
    "is_signer":true,
    "block_pub":"'$GENERATOR_PUBKEY'",    
    "quorum":2,
    "block_signer_urls":[{
        "pubkey":"'$SIGNER1_PUBKEY'",   
        "url":"'$SIGNER1_URL'"        
    },{
        "pubkey":"'$SIGNER2_PUBKEY'",
        "url":"'$SIGNER2_URL'"
    }]
}'
```

#### Retrieve the blockchain id
```
$ BLOCKCHAIN_ID=`curl --silent $GENERATOR_URL/info | jq -r .blockchain_id`
```

#### Configure the signers
Signer 1:
```
$ curl --silent $SIGNER1_PUBKEY/configure --data '{
    "is_signer":true,
    "blockchain_id":"'$BLOCKCHAIN_ID'",
    "generator_url":"'$GENERATOR_URL'",
    "block_pub":"'$SIGNER1_PUBKEY'"
}'
```

Signer 2:
```
$ curl --silent $SIGNER2_PUBKEY/configure --data '{
    "is_signer":true,
    "blockchain_id":"'$BLOCKCHAIN_ID'",
    "generator_url":"'$GENERATOR_URL'",
    "block_pub":"'$SIGNER2_PUBKEY'"
}'
```

### Authenticated
Creating an authenticated network makes use of client
and network access tokens. Follow the steps from earlier
to define your configuration variables and create block
signing keypairs. The rest of this guide will walk you
through creating client/network access tokens and
calling the modified configuration commands that use
them.


#### Create client access tokens
Generator:
```
$ GENERATOR_CLIENT_TOKEN=`DATABASE_URL=$GENERATOR_DB_URL corectl create-token client`
```

Signer 1:
```
$ SIGNER1_CLIENT_TOKEN=`DATABASE_URL=$SIGNER1_DB_URL corectl create-token client`
```

Signer 2:
```
$ SIGNER2_CLIENT_TOKEN=`DATABASE_URL=$SIGNER2_DB_URL corectl create-token client`
```

#### Create network access tokens
Run this command passing each core's database url:
Generator:
```
$ GENERATOR_NETWORK_TOKEN=`DATABASE_URL=$GENERATOR_DB_URL corectl create-token -net network`
```

Signer 1:
```
$ SIGNER1_NETWORK_TOKEN=`DATABASE_URL=$SIGNER1_DB_URL corectl create-token -net network`
```

Signer 2:
```
$ SIGNER2_NETWORK_TOKEN=`DATABASE_URL=$SIGNER2_DB_URL corectl create-token -net network`
```

#### Configure the generator
```
$ curl --silent --user $GENERATOR_CLIENT_TOKEN $GENERATOR_URL/configure --data '{
    "is_generator":true,
    "is_signer":true,
    "block_pub":"'$GENERATOR_PUBKEY'",    
    "quorum":2,
    "block_signer_urls":[{
        "pubkey":"'$SIGNER1_PUBKEY'",   
        "url":"'$SIGNER1_URL'",
        "access_token":"'$SIGNER1_NETWORK_TOKEN'"
    },{
        "pubkey":"'$SIGNER2_PUBKEY'",
        "url":"'$SIGNER2_URL'",
        "access_token":"'$SIGNER2_NETWORK_TOKEN'"
    }]
}'
```

#### Retrieve the blockchain id
```
$ BLOCKCHAIN_ID=`curl --silent --user $GENERATOR_CLIENT_TOKEN $GENERATOR_URL/info | jq -r .blockchain_id`
```

#### Configure the signers
Signer 1:
```
$ curl --silent --user $SIGNER1_CLIENT_TOKEN $SIGNER1_PUBKEY/configure --data '{
    "is_signer":true,
    "blockchain_id":"'$BLOCKCHAIN_ID'",
    "generator_url":"'$GENERATOR_URL'",
    "block_pub":"'$SIGNER1_PUBKEY'",
    "generator_access_token":"'$GENERATOR_NETWORK_TOKEN'"
}'
```

Signer 2:
```
$ curl --silent --user $SIGNER2_CLIENT_TOKEN $SIGNER2_PUBKEY/configure --data '{
    "is_signer":true,
    "blockchain_id":"'$BLOCKCHAIN_ID'",
    "generator_url":"'$GENERATOR_URL'",
    "block_pub":"'$SIGNER2_PUBKEY'",
    "generator_access_token":"'$GENERATOR_NETWORK_TOKEN'"
}'
```

## Building a new release

1. Update `/docs/install.md` for impending verson
   change.
2. Build and bundle most recent dashboard:
   `bin/bundle-dashboard`
3. Commit bundle changes in `$CHAIN/generated` to `main`
4. Prepare new installer apps based on `main`, using a
   `latest` namescheme.
    - TODO: per-platform instructions
5. Upload installer apps to s3://download.chain.com
    - TODO: automate this uploader
6. `bin/upload-docs` - This will build and upload the
   docs.
    - By default, documentation is uploaded to http://chain-staging.chain.com. To
      upload to the production site, run `bin/upload-docs prod`.

TODO: Determine whether we should also use GitHub's
releases feature.
