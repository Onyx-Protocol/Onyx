# Operations

* [Creating Networks](#creating-networks)
  * [Unauthenticated](#unauthenticated)
  * [Authenticated](#authenticated)

## Creating Networks
### Unauthenticated
This guide will demonstrate creating a 2-of-3 block signing network with a generator and two signers.
We assume you have database access to 3 running, unconfigured cores. We will also makes use of the `corectl`, `curl`
and `jq` commands. Please note that Chain Core only allows unauthenticated connections on the loopback interface. If you
will be connecting to the cores through another network device, please follow the instructions [here](#authenticated).

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
Creating an authenticated network makes use of client and network access tokens. Follow the steps from earlier
to define your configuration variables and create block signing keypairs. The rest of this guide will walk you
through creating client/network access tokens and calling the modified configuration commands that use them.


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

