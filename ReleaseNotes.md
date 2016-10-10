Release notes for Chain Core
============================

## Release of 24 Oct 2016

- Expert third-party cryptanalysis of the derived-key scheme in chain/crypto/chainkd is in progress. While review is still under way, do not rely on this algorithm for security or compatibility: it is subject to change as a result of review.
- The mechanism for altering the consensus program in a running network is still in development. This means that once a new network is configured, its set of block-signing keys is fixed.
- The detached-block-signing flow does not yet include [the specification step](https://github.com/chain/cp1/blob/main/consensus.md#sign-block) in which the signer verifies the generator's signature of the block.
