# Virtual Machine 2 Specification

* [Introduction](#introduction)
* [Summary of changes](#summary-of-changes)
* [Amended instructions](#amended-instructions)
* [Additional instructions](#additional-instructions)
* [Expansion opcodes](#expansion-opcodes)
* [References](#references)


## Introduction

The version 2 of the Chain Virtual Machine is an extension of [VM 1](vm1.md) designed to support [Confidential Assets](ca.md).

Present document specifies only the difference with VM 1, please see the [VM 1 specification](vm1.md) for more details.

## Summary of changes

VM2 has the similar design to VM1, differing only in additional instructions and amended semantics for some existing instructions.

* VM2 is only relevant in transaction context and never used in a block context.
* VM2 adds additional introspection instructions to support [Confidential Assets](ca.md).
* VM2 amends several VM1 introspection instructions to support [Confidential Assets](ca.md).
* VM2 is required to be used in [CA](ca.md) entries with an exception for [Asset Issuance Choice](blockchain.md#asset-issuance-choice) where VM1 is supported to enable confidential issuance of the existing asset IDs.
* VM2 can be used in all entries, including pre-CA entries. This is to allow pre-CA accounts to make payments to post-CA accounts and enable programmatic [upgrades](#upgrade).


## Amended instructions

#### CHECKPREDICATE

Code  | Stack Diagram            | Cost
------|--------------------------|----------------------------
0xc0  | (n predicate limit → q)  | 256 + limit; [standard memory cost](#standard-memory-cost) – 256 + 64 – leftover

[Same behaviour](vm1.md#checkpredicate) as in VM1, except it instantiates VM2 instead of VM1 for predicate evaluation.

#### VERIFYISSUANCEKEY

Code  | Stack Diagram               | Cost
------|-----------------------------|-----------------------------------------------------
0xab  | (issuancekey → issuancekey) | 1; [standard memory cost](#standard-memory-cost)

`VERIFYISSUANCEKEY` is an expansion opcode [specified](vm1.md#verifyissuancekey) in VM1 to enforce an issuance key in [Asset Issuance Choice](blockchain.md#asset-issuance-choice). 

In VM2 it has the [same](vm1.md#verifyissuancekey) behavior as in VM1.

#### CHECKOUTPUT

Code  | Stack Diagram                                        | Cost
------|------------------------------------------------------|-----------------------------------------------------
0xc1  | (index data amount assetid version prog → q)         | 16; [standard memory cost](#standard-memory-cost)

Compared to [CHECKOUTPUT](vm1.md#checkoutput) in VM1, in VM2 this instruction uses [value commitments](ca.md#value-commitment) instead of a cleartext amounts and [asset ID commitments](ca.md#asset-id-commitment) instead of cleartext [asset IDs](blockchain.md#asset-id) when evaluating in the context of [Mux2](blockchain.md#mux-2), [Issuance2](blockchain.md#issuance-2) or [Spend2](blockchain.md#spend-2).

In both version 1 and version 2 entries, `CHECKOUTPUT` in VM2 is not used to match retirement entries, for that use [CHECKRETIREMENT](#checkretirement).

1. If the current entry is one of [Mux1](blockchain.md#mux-1), [Issuance1](blockchain.md#issuance-1) or [Spend1](blockchain.md#spend-1):
    1. Pops 6 items from the data stack: `index`, `data`, `amount`, `assetid`, `version`, `prog`.
    2. Fails if `index` is negative or not a valid [number](vm1.md#vm-number).
    3. Fails if `version` is not a [number](vm1.md#vm-number).
    4. Fails if `amount` is not a valid [number](vm1.md#vm-number).
    5. Fails if `assetid` is not a 32-byte [string](vm1.md#vm-string).
    6. If the current entry is a [Mux1](blockchain.md#mux-1):
        1. Finds a [destination entry](blockchain.md#value-destination-1) at the given `index`. Fails if there is no entry at `index`.
        2. If the entry satisfies all of the following conditions pushes [true](vm1.md#vm-boolean) on the data stack; otherwise pushes [false](vm1.md#vm-boolean):
            1. destination entry type is [Output1](blockchain.md#output-1),
            2. output VM version equals `version`,
            3. output program bytecode equals `prog`,
            4. output asset ID commitment equals `assetid`,
            5. output value commitment equals `amount`,
            6. `data` is an empty string or it matches the 32-byte data string in the output.
    7. If the entry is an [Issuance1](blockchain.md#issuance-1) or a [Spend1](blockchain.md#spend-1):
        1. If the [destination entry](blockchain.md#value-destination-1) is a [Mux1](blockchain.md#mux-1), performs checks as described in step 6.
        2. If the [destination entry](blockchain.md#value-destination-1) is an [Output1](blockchain.md#output-1):
            1. If `index` is not zero, pushes [false](vm1.md#vm-boolean) on the data stack.
            2. Otherwise, performs checks as described in step 6.2.
2. If the current entry is one of [Mux2](blockchain.md#mux-2), [Issuance2](blockchain.md#issuance-2) or [Spend2](blockchain.md#spend-2):
    1. Pops 6 items from the data stack: `index`, `data`, `amount`, `assetid`, `version`, `prog`.
    2. Fails if `index` is negative or not a valid [number](vm1.md#vm-number).
    3. Fails if `version` is not a [number](vm1.md#vm-number).
    4. Fails if `amount` is not a valid [point pair](ca.md#point-pair) (representing a [value commitment](ca.md#value-commitment)).
    5. Fails if `assetid` is not a valid [point pair](ca.md#point-pair) (representing an [asset ID commitment](ca.md#asset-id-commitment)).
    6. If the current entry is a [Mux2](blockchain.md#mux-2):
        1. Finds a [destination entry](blockchain.md#value-destination-2) at the given `index`. Fails if there is no entry at `index`.
        2. If the entry satisfies all of the following conditions pushes [true](vm1.md#vm-boolean) on the data stack; otherwise pushes [false](vm1.md#vm-boolean):
            1. destination entry type is [Output2](blockchain.md#output-2),
            2. output VM version equals `version`,
            3. output program bytecode equals `prog`,
            4. output asset ID commitment equals `assetid`,
            5. output value commitment equals `amount`,
            6. `data` is an empty string or it matches the 32-byte data string in the output.
    7. If the entry is an [Issuance2](blockchain.md#issuance-2) or a [Spend2](blockchain.md#spend-2):
        1. If the [destination entry](blockchain.md#value-destination-2) is a [Mux2](blockchain.md#mux-2), performs checks as described in step 6.
        2. If the [destination entry](blockchain.md#value-destination-2) is an [Output2](blockchain.md#output-2):
            1. If `index` is not zero, pushes [false](vm1.md#vm-boolean) on the data stack.
            2. Otherwise, performs checks as described in step 6.2.

Fails if executed in the [block context](#block-context).

Fails if the entry is not a [Mux1](blockchain.md#mux-1)/[Mux2](blockchain.md#mux-2), [Issuance1](blockchain.md#issuance-1)/[Issuance2](blockchain.md#issuance-2) or [Spend1](blockchain.md#spend-1)/[Spend2](blockchain.md#spend-2).


#### ASSET

Code  | Stack Diagram  | Cost
------|----------------|-----------------------------------------------------
0xc2  | (∅ → assetid)  | 1; [standard memory cost](#standard-memory-cost)

1. If the current entry is an [Issuance1](blockchain.md#issuance-1), pushes `Value.AssetID`.
2. If the current entry is a [Spend1](blockchain.md#spend-1), pushes the `SpentOutput.Source.Value.AssetID` of that entry.
3. If the current entry is an [Issuance2](blockchain.md#issuance-2):
    2. If the [confidential issuance choice flag](#vm-state) is off (VM evaluates an issuance delegate program):
        1. Verifies that the [issuance asset range proof](ca.md#non-confidential-issuance-asset-range-proof) is non-confidential.
        2. Pushes the `Value.AssetID` of that issuance entry.
    3. If the [confidential issuance choice flag](#vm-state) is on (VM evaluates an issuance program for an asset ID choice):
        1. Pushes the `AssetID` of that issuance choice.
4. If the current entry is a [Spend2](blockchain.md#spend-2):
    1. Verifies that the [asset range proof](ca.md#non-confidential-asset-range-proof) is non-confidential.
    2. Pushes the `SpentOutput.Source.Value.AssetID` of that entry.
5. If the current entry is a [Nonce](blockchain.md#nonce) entry:
    1. Verifies that the `AnchoredEntry` reference is an [Issuance 1](blockchain.md#issuance-1) or an [Issuance 2](blockchain.md#issuance-2) entry.
    2. If `AnchoredEntry` is an [Issuance 1](blockchain.md#issuance-1), pushes the `Value.AssetID` of that issuance entry.
    3. If `AnchoredEntry` is an [Issuance 2](blockchain.md#issuance-2):
        1. Verifies that the [issuance asset range proof](ca.md#non-confidential-issuance-asset-range-proof) is non-confidential.
        2. Pushes the `Value.AssetID` of that issuance entry.

Fails if executed in the [block context](#block-context).

Fails if the entry is not a [Nonce](blockchain.md#nonce), an [Issuance1](blockchain.md#issuance-1)/[Issuance2](blockchain.md#issuance-2) or a [Spend1](blockchain.md#spend-1)/[Spend2](blockchain.md#spend-2).

#### AMOUNT

Code  | Stack Diagram  | Cost
------|----------------|-----------------------------------------------------
0xc3  | (∅ → amount)   | 1; [standard memory cost](#standard-memory-cost)

1. If the current entry is an [Issuance1](blockchain.md#issuance-1), pushes `Value.Amount`.
2. If the current entry is a [Spend1](blockchain.md#spend-1), pushes the `SpentOutput.Source.Value.Amount` of that entry.
3. If the current entry is an [Issuance2](blockchain.md#issuance-2):
    1. If the [confidential issuance choice flag](#vm-state) is off (VM evaluates an issuance delegate program):
        1. Verifies that the [issuance asset range proof](ca.md#non-confidential-issuance-asset-range-proof) is non-confidential.
        2. Pushes the `Value.Amount` of the issuance entry.
    2. If the [confidential issuance choice flag](#vm-state) is on (VM evaluates an issuance program for an asset ID choice):
        1. Verifies that the [issuance asset range proof](ca.md#non-confidential-issuance-asset-range-proof) is non-confidential.
        2. Pushes the `Value.Amount` of the issuance entry.
4. If the current entry is a [Spend2](blockchain.md#spend-2):
    1. Verifies that the [asset range proof](ca.md#non-confidential-asset-range-proof) is non-confidential.
    2. Pushes the `SpentOutput.Source.Value.Amount` of that entry.
5. If the current entry is a [Nonce](blockchain.md#nonce) entry:
    1. Verifies that the `AnchoredEntry` reference is an [Issuance 1](blockchain.md#issuance-1) or an [Issuance 2](blockchain.md#issuance-2) entry.
    2. If `AnchoredEntry` is an [Issuance 1](blockchain.md#issuance-1), pushes the `Value.Amount` of that issuance entry.
    3. If `AnchoredEntry` is an [Issuance 2](blockchain.md#issuance-2):
        1. Verifies that the [issuance asset range proof](ca.md#non-confidential-issuance-asset-range-proof) is non-confidential.
        2. Pushes the `Value.Amount` of that issuance entry.

Fails if executed in the [block context](#block-context).

Fails if the entry is not a [Nonce](blockchain.md#nonce), an [Issuance1](blockchain.md#issuance-1)/[Issuance2](blockchain.md#issuance-2) or a [Spend1](blockchain.md#spend-1)/[Spend2](blockchain.md#spend-2).


#### PROGRAM

Code  | Stack Diagram  | Cost
------|----------------|-----------------------------------------------------
0xc4  | (∅ → program)   | 1; [standard memory cost](#standard-memory-cost)

Pushes a program based on a type of current entry and the context:

Entry Type                                               | Program
---------------------------------------------------------|----------------
[Nonce](blockchain.md#nonce)                             | Nonce program
[Issuance1](blockchain.md#issuance-1)                    | Issuance program for the asset ID
[Issuance2](blockchain.md#issuance-2) (delegate program) | Issuance delegate program as specified in the issuance entry
[Issuance2](blockchain.md#issuance-2) (issuance choice)  | Issuance program for a given [asset ID choice](#asset-issuance-choice)
[Spend1](blockchain.md#spend-1)                          | Control program of the output being spent
[Spend2](blockchain.md#spend-2)                          | Control program of the output being spent
[Mux1](blockchain.md#mux-1)                              | Mux program
[Mux2](blockchain.md#mux-2)                              | Mux program

Fails if executed in the [block context](#block-context).


## Additional instructions

#### CHECKRETIREMENT

Code  | Stack Diagram                                        | Cost
------|------------------------------------------------------|-----------------------------------------------------
0xcf  | (index data amount assetid → q)                      | 16; [standard memory cost](#standard-memory-cost)

Checks that the value is retired. 

This instruction uses [value commitments](ca.md#value-commitment) instead of a cleartext amounts and [asset ID commitments](ca.md#asset-id-commitment) instead of cleartext [asset IDs](blockchain.md#asset-id) when evaluating in the context of [Mux2](blockchain.md#mux-2), [Issuance2](blockchain.md#issuance-2) or [Spend2](blockchain.md#spend-2).

1. If the current entry is one of [Mux1](blockchain.md#mux-1), [Issuance1](blockchain.md#issuance-1) or [Spend1](blockchain.md#spend-1):
    1. Pops 4 items from the data stack: `index`, `data`, `amount`, `assetid`.
    2. Fails if `index` is negative or not a valid [number](vm1.md#vm-number).
    3. Fails if `amount` is not a valid [number](vm1.md#vm-number).
    4. Fails if `assetid` is not a 32-byte [string](vm1.md#vm-string).
    5. If the current entry is a [Mux1](blockchain.md#mux-1):
        1. Finds a [destination entry](blockchain.md#value-destination-1) at the given `index`. Fails if there is no entry at `index`.
        2. If the entry satisfies all of the following conditions pushes [true](vm1.md#vm-boolean) on the data stack; otherwise pushes [false](vm1.md#vm-boolean):
            1. destination entry type is [Retirement1](blockchain.md#output-1),
            2. retirement asset ID equals `assetid`,
            3. retirement amount equals `amount`,
            4. retirement upgrade program’s VM version equals 0,
            5. `data` is an empty string or it matches the 32-byte data string in the output.
    6. If the entry is an [Issuance1](blockchain.md#issuance-1) or a [Spend1](blockchain.md#spend-1):
        1. If the [destination entry](blockchain.md#value-destination-1) is a [Mux1](blockchain.md#mux-1), performs checks as described in step 5.
        2. If the [destination entry](blockchain.md#value-destination-1) is an [Retirement1](blockchain.md#retirement-1):
            1. If `index` is not zero, pushes [false](vm1.md#vm-boolean) on the data stack.
            2. Otherwise, performs checks as described in step 5.2.
2. If the current entry is one of [Mux2](blockchain.md#mux-2), [Issuance2](blockchain.md#issuance-2) or [Spend2](blockchain.md#spend-2):
    1. Pops 4 items from the data stack: `index`, `data`, `amount`, `assetid`.
    2. Fails if `index` is negative or not a valid [number](vm1.md#vm-number).
    3. Fails if `amount` is not a valid [point pair](ca.md#point-pair) (representing a [value commitment](ca.md#value-commitment)).
    4. Fails if `assetid` is not a valid [point pair](ca.md#point-pair) (representing an [asset ID commitment](ca.md#asset-id-commitment)).
    5. If the current entry is a [Mux2](blockchain.md#mux-2):
        1. Finds a [destination entry](blockchain.md#value-destination-2) at the given `index`. Fails if there is no entry at `index`.
        2. If the entry satisfies all of the following conditions pushes [true](vm1.md#vm-boolean) on the data stack; otherwise pushes [false](vm1.md#vm-boolean):
            1. destination entry type is [Retirement2](blockchain.md#retirement-2),
            2. retirement asset ID commitment equals `assetid`,
            3. retirement value commitment equals `amount`,
            4. `data` is an empty string or it matches the 32-byte data string in the output.
    6. If the entry is an [Issuance2](blockchain.md#issuance-2) or a [Spend2](blockchain.md#spend-2):
        1. If the [destination entry](blockchain.md#value-destination-2) is a [Mux2](blockchain.md#mux-2), performs checks as described in step 5.
        2. If the [destination entry](blockchain.md#value-destination-2) is an [Retirement2](blockchain.md#retirement-2):
            1. If `index` is not zero, pushes [false](vm1.md#vm-boolean) on the data stack.
            2. Otherwise, performs checks as described in step 5.2.

Fails if executed in the [block context](#block-context).

Fails if the entry is not a [Mux1](blockchain.md#mux-1)/[Mux2](blockchain.md#mux-2), [Issuance1](blockchain.md#issuance-1)/[Issuance2](blockchain.md#issuance-2) or [Spend1](blockchain.md#spend-1)/[Spend2](blockchain.md#spend-2).

#### VERIFYUPGRADE

Code  | Stack Diagram                                                    | Cost
------|------------------------------------------------------------------|-----------------------------------------------------
0xd0  | (index data amount assetid vmversion program upgradetype → ∅)    | 16; [standard memory cost](#standard-memory-cost)

Checks that the value is upgraded. 

This instruction uses [value commitments](ca.md#value-commitment) instead of a cleartext amounts and [asset ID commitments](ca.md#asset-id-commitment) instead of cleartext [asset IDs](blockchain.md#asset-id) when evaluating in the context of [Mux2](blockchain.md#mux-2), [Issuance2](blockchain.md#issuance-2) or [Spend2](blockchain.md#spend-2).

1. If the current entry is one of [Mux1](blockchain.md#mux-1), [Issuance1](blockchain.md#issuance-1) or [Spend1](blockchain.md#spend-1):
    1. Pops 7 items from the data stack: `index`, `data`, `amount`, `assetid`, `vmversion`, `program` and `upgradetype`.
    2. Fails if `index` is negative or not a valid [number](vm1.md#vm-number).
    3. Fails if `amount` and `vmversion` are not valid [numbers](vm1.md#vm-number).
    4. Fails if `assetid` is not a 32-byte [string](vm1.md#vm-string).
    5. If the current entry is a [Mux1](blockchain.md#mux-1):
        1. Finds a [destination entry](blockchain.md#value-destination-1) at the given `index`. Fails if there is no entry at `index`.
        2. If the entry satisfies all of the following conditions pushes [true](vm1.md#vm-boolean) on the data stack; otherwise pushes [false](vm1.md#vm-boolean):
            1. destination entry type is [Retirement1](blockchain.md#output-1),
            2. retirement asset ID equals `assetid`,
            3. retirement amount equals `amount`,
            4. retirement upgrade program’s VM version equals `vmversion`,
            5. `data` is an empty string or it matches the 32-byte data string in the output.
            6. verify that retirement’s `UpgradeDestination` points to a valid entry with type `upgradetype`.
    6. If the entry is an [Issuance1](blockchain.md#issuance-1) or a [Spend1](blockchain.md#spend-1):
        1. If the [destination entry](blockchain.md#value-destination-1) is a [Mux1](blockchain.md#mux-1), performs checks as described in step 5.
        2. If the [destination entry](blockchain.md#value-destination-1) is an [Retirement1](blockchain.md#output-1):
            1. If `index` is not zero, pushes [false](vm1.md#vm-boolean) on the data stack.
            2. Otherwise, performs checks as described in step 5.2.
2. If the current entry is one of [Mux2](blockchain.md#mux-2), [Issuance2](blockchain.md#issuance-2) or [Spend2](blockchain.md#spend-2):
    1. Pops 4 items from the data stack: `index`, `data`, `amount`, `assetid`.
    2. Fails if `index` is negative or not a valid [number](vm1.md#vm-number).
    3. Fails if `vmversion` is not a valid [number](vm1.md#vm-number).
    4. Fails if `amount` is not a valid [point pair](ca.md#point-pair) (representing a [value commitment](ca.md#value-commitment)).
    5. Fails if `assetid` is not a valid [point pair](ca.md#point-pair) (representing an [asset ID commitment](ca.md#asset-id-commitment)).
    6. If the current entry is a [Mux2](blockchain.md#mux-2):
        1. Finds a [destination entry](blockchain.md#value-destination-2) at the given `index`. Fails if there is no entry at `index`.
        2. If the entry satisfies all of the following conditions pushes [true](vm1.md#vm-boolean) on the data stack; otherwise pushes [false](vm1.md#vm-boolean):
            1. destination entry type is [Retirement2](blockchain.md#retirement-2),
            2. retirement asset ID commitment equals `assetid`,
            3. retirement value commitment equals `amount`,
            4. `data` is an empty string or it matches the 32-byte data string in the output.
            5. If the [expansion flag](#vm-state) is on: do nothing.
            6. If the [expansion flag](#vm-state) is off: fail execution.
    7. If the entry is an [Issuance2](blockchain.md#issuance-2) or a [Spend2](blockchain.md#spend-2):
        1. If the [destination entry](blockchain.md#value-destination-2) is a [Mux2](blockchain.md#mux-2), performs checks as described in step 6.
        2. If the [destination entry](blockchain.md#value-destination-2) is an [Output2](blockchain.md#output-2):
            1. If `index` is not zero, pushes [false](vm1.md#vm-boolean) on the data stack.
            2. Otherwise, performs checks as described in step 6.2.

Fails if executed in the [block context](#block-context).

Fails if the entry is not a [Mux1](blockchain.md#mux-1)/[Mux2](blockchain.md#mux-2), [Issuance1](blockchain.md#issuance-1)/[Issuance2](blockchain.md#issuance-2) or [Spend1](blockchain.md#spend-1)/[Spend2](blockchain.md#spend-2).


#### MAKECOMMITMENT

Code  | Stack Diagram                   | Cost
------|---------------------------------|-----------------------------------------------------
0xd1  | (amount assetid → commitment)   | 32; [standard memory cost](#standard-memory-cost)

Pushes the non-blinded [value commitment](ca.md#value-commitment) encoded as a 64-byte string on the data stack.

Note: in order to create an [asset ID commitment](ca.md#asset-id-commitment), use this instruction with `amount` set to 1.

Fails if executed in the [block context](#block-context).


#### ASSETCOMMITMENT

Code  | Stack Diagram           | Cost
------|-------------------------|-----------------------------------------------------
0xd2  | (∅ → assetcommitment)   | 1; [standard memory cost](#standard-memory-cost)

Pushes the [asset ID commitment](ca.md#asset-id-commitment) encoded as a 64-byte string on the data stack.

1. If the current entry is an [Issuance1](blockchain.md#issuance-1), fails execution.
2. If the current entry is a [Spend1](blockchain.md#spend-1), fails execution.
3. If the current entry is an [Issuance2](blockchain.md#issuance-2):
    1. If the [confidential issuance choice flag](#vm-state) is off (VM evaluates an issuance delegate program):
        1. Pushes asset ID commitment specified in the issuance entry.
    2. If the [confidential issuance choice flag](#vm-state) is on (VM evaluates an issuance program for an asset ID choice):
        1. Converts `AssetID` in the issuance choice as a non-confidential asset ID commitment and pushes it on stack.
4. If the current entry is a [Spend2](blockchain.md#spend-2):
    1. Pushes the `SpentOutput.Source.Value.AssetIDCommitment` of that entry.
5. If the current entry is a [Nonce](blockchain.md#nonce) entry:
    1. Verifies that the `AnchoredEntry` reference is an [Issuance 2](blockchain.md#issuance-2) entry.
    2. Pushes the `Value.AssetIDCommitment` of that issuance entry.

Fails if executed in the [block context](#block-context).


#### VALUECOMMITMENT

Code  | Stack Diagram             | Cost
------|---------------------------|-----------------------------------------------------
0xd3  | (∅ → valuecommitment)     | 1; [standard memory cost](#standard-memory-cost)

Pushes the [value commitment](ca.md#value-commitment) encoded as a 64-byte string on the data stack.

1. If the current entry is an [Issuance1](blockchain.md#issuance-1), fails execution.
2. If the current entry is a [Spend1](blockchain.md#spend-1), fails execution
3. If the current entry is an [Issuance2](blockchain.md#issuance-2):
    1. If the [confidential issuance choice flag](#vm-state) is off (VM evaluates an issuance delegate program):
        1. Pushes the `Value.ValueCommitment` of the issuance entry.
    2. If the [confidential issuance choice flag](#vm-state) is on (VM evaluates an issuance program for an asset ID choice):
        1. Pushes the `Value.ValueCommitment` of the issuance entry.
4. If the current entry is a [Spend2](blockchain.md#spend-2):
    1. Pushes the `SpentOutput.Source.Value.ValueCommitment` of that entry.
5. If the current entry is a [Nonce](blockchain.md#nonce) entry:
    1. Verifies that the `AnchoredEntry` reference is an [Issuance 2](blockchain.md#issuance-2) entry.
    2. Pushes the `Value.ValueCommitment` of that issuance entry.

Fails if executed in the [block context](#block-context).


#### ADDCOMMITMENT

Code  | Stack Diagram   | Cost
------|-----------------|----------------------------
0xd4  | (A B → A+B)     | 16; [standard memory cost](#standard-memory-cost)

Pops two [point pairs](ca.md#point-pair) from the data stack, adds them, and pushes the result to the data stack.

Fails if `A` or `B` is not a valid [point pair](ca.md#point-pair).


#### SUBCOMMITMENT

Code  | Stack Diagram   | Cost
------|-----------------|----------------------------
0xd5  | (A B → A–B)     | 16; [standard memory cost](#standard-memory-cost)

Pops two [point pairs](ca.md#point-pair) from the data stack, subtracts them, and pushes the result to the data stack.

Fails if `A` or `B` is not a valid [point pair](ca.md#point-pair).


#### MULCOMMITMENT

Code  | Stack Diagram   | Cost
------|-----------------|----------------------------
0xd6  | (A b → b·A)     | 512; [standard memory cost](#standard-memory-cost)

1. Pops a [number](#vm-number) from the data stack.
2. Pops a [point pair](ca.md#point-pair) from the data stack.
3. [Multiplies](ca.md#point-operations) the point pair `A` with a number `b` and pushes the resulting point pair to the data stack.

Fails if `A` is not a valid [point pair](ca.md#point-pair).

Fails if `b` is not a valid [VM number](#vm-number).




## Expansion opcodes

Code  | Stack Diagram   | Cost
------|-----------------|-----------------------------------------------------
0x50, 0x61, 0x62, 0x65, 0x66, 0x67, 0x68, 0x8a, 0x8d, 0x8e, 0xa6, 0xa7, 0xa9, 0xab, 0xb0..0xbf, 0xd7..0xff  | (∅ → ∅)     | 1

The unassigned codes are reserved for future expansion.

If the [expansion flag](#vm-state) is on, these opcodes have no effect on the state of the VM except from reducing run limit by 1 and incrementing the program counter.

If the [expansion flag](#vm-state) is off, these opcodes immediately fail the program when encountered during execution.





# References

* [FIPS180: "Secure Hash Standard", United States of America, National Institute of Standards and Technology, Federal Information Processing Standard 180-2](http://csrc.nist.gov/publications/fips/fips180-2/fips180-2withchangenotice.pdf).
* [FIPS202: Federal Inf. Process. Stds. (NIST FIPS) - 202 (SHA3)](https://dx.doi.org/10.6028/NIST.FIPS.202)
* [LEB128: Little-Endian Base-128 Encoding](https://developers.google.com/protocol-buffers/docs/encoding)
* [RFC 6962](https://tools.ietf.org/html/rfc6962#section-2.1)
* [RFC 8032](https://tools.ietf.org/html/rfc8032)
