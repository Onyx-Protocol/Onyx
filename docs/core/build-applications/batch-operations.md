# Batch Operations

## Overview

Batch operations are an advanced feature of the Chain Core API that allow you to bundle many similar operations into a single API call. Not only does this reduce network overhead, it also allows Chain Core to process the operations in your request in parallel.

Operations that support batching include:

* Creating assets
* Creating accounts
* Creating control programs
* Building transactions
* Signing transactions
* Submitting transactions

### Sample Code

All code samples in this guide can be viewed in a single, runnable script. Available languages:

- [Java](../examples/java/BatchOperations.java)
- [Ruby](../examples/ruby/batch_operations.rb)

## Example: Creating assets in a batch

All batch operations share a common workflow. To illustrate this, we’ll walk through an example program that creates several assets as a batch.

#### Preparing the request

Batch operations take similar input to their non-batch analogs, except that in batch calls you pass a _list_ of parameter objects, rather than a single object.

For our example, we’ll create a list of builder objects for our assets, one for each asset we want to create:

$code asset-builders ../examples/java/BatchOperations.java ../examples/ruby/batch_operations.rb

Note that we’re attempting to create the `bronze` asset with an invalid value for the `quorum` parameter. This will generate an API error, and since we’re performing a batch operation, we’ll have to handle the error in a special way.

#### Making the batch call

All batch method names in the SDK end with `Batch`, and they return a special batch response object:

$code asset-create-batch ../examples/java/BatchOperations.java ../examples/ruby/batch_operations.rb

#### Handling the response

If there is a problem affecting the entire batch request, such as a network error, then the SDK will ensure that an exception is thrown. However, if there is a problem with an _individual_ item in your batch request, no exception will be thrown. Some of the items in your batch request may have succeeded, while others might have failed.

The `BatchResponse` object provides an easy interface for determining which items in the request successed, and which ones resulted in an error. We can iterate over the items in the batch response (there is one for every parameter object in the original request) and print out whether the operation succeeded or failed.

$code asset-create-handle-errors ../examples/java/BatchOperations.java ../examples/ruby/batch_operations.rb

In our example, we expect at least one of the items in the batch request to fail, since we provided an invalid value for the `bronze` asset’s `quorum` parameter. This produces the following output:

```
asset 0 created, ID: f7cfb9604bc53b2ad44d2b61764cf77b4ffbcc16fd25206ce1ff4c2022f914dd
asset 1 created, ID: 25217326f04305cbd83e809eeb95d325b4e6bacc30aa7d46a08406bc4e20579a
asset 2 error: com.chain.exception.APIException: Code: CH200 Message: Quorum must be greater than 1 and less than or equal to the length of xpubs
```

#### Parallelization and errors

Batch operations are parallelized on the server side, so there may be some non-deterministic error behavior if items in your batch request conflict with each other. Consider the following example:

$code nondeterministic-errors ../examples/java/BatchOperations.java ../examples/ruby/batch_operations.rb

Here, all three asset builders are attempting to create an asset with the alias `platinum`. Aliases must be unique, so only one will succeed. Due to parallelization, it’s possible to get results that appear to be out-of-order relative to the original request. For example, it’s possible for the last item in the request to succeed while the first two produce errors:

```
asset 0 error: com.chain.exception.APIException: Code: CH003 Message: Invalid request body Detail: non-unique alias
asset 1 error: com.chain.exception.APIException: Code: CH003 Message: Invalid request body Detail: non-unique alias
asset 2 created, ID: 9f7e068bf207faf60f08e78f8ae2834ac35340c217ee2b2329fe25724f246f42
```

## Example: Batch transactions

Each of the three primary steps of transacting in Chain Core--building, signing, and submitting--can be performed as a batch operation. We’ll experiment with this by attempting to issue three different assets to Alice in three separate transactions as a batch.

#### Building

First, we’ll put together a list of transaction builders:

$code batch-build-builders ../examples/java/BatchOperations.java ../examples/ruby/batch_operations.rb

The second transaction (index `1`) in our list is attempting to issue a non-existent asset, so we expect it to fail. If we make the batch request, we can iterate over the response errors and display any errors.

$code batch-build-handle-errors ../examples/java/BatchOperations.java ../examples/ruby/batch_operations.rb

This error handling loop should produce the following output:

```
Error building transaction 1: com.chain.exception.APIException: Code: CH002 Message: Not found Detail: invalid asset alias not-a-real-asset on action 0
```

#### Signing

Let’s move on and try to sign and submit the transactions that we successfully built. We can extract the successful responses using the `successes()` method and pass them to `signBatch`:

$code batch-sign ../examples/java/BatchOperations.java ../examples/ruby/batch_operations.rb

#### Submitting

Assuming there are no errors in signing, we can submit our batch of transactions:

$code batch-submit ../examples/java/BatchOperations.java ../examples/ruby/batch_operations.rb

Finally, assuming there are no errors during submission, we should see the following output:

```
Transaction 0 submitted, ID: 25dea2cb088586b46184556ffedc4c481b5e34b28e6f6f176731a9aecd1286ea
Transaction 1 submitted, ID: 10666b27f303ed52f90381edd945fff5298251867dc398e7fe4bc3e2d5ccba52
```
