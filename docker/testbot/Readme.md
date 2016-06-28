#Testbot
##Introduction
This container runs the integration test suite. Tests include a singlecore suite and multi-core tests with the following block signing configurations:
- 1-of-1 sigs required: 1 generator (non-signer), 1 remote signer
- 1-of-2 sigs required: 1 generator+signer, 1 remote signer
- 1-of-2 sigs required: 1 generator (non-signer), 2 remote signers
- 2-of-2 sigs required: 1 generator+signer, 1 signer
- 2-of-2 sigs required: 1 generator (non-signer), 2 remote signers

##Quickstart
####Setup environment
- Set `CHAIN` to chain src directory
- Set `PRIV1` to an ECDSA priv key
- Set `PUB1` to `PRIV1`'s corresponding pub key
- Set `PRIV2` to an ECDSA priv key
- Set `PUB2` to `PRIV2`'s corresponding pub key

####Run the tests
```
$ sh bin/run-tests
```
