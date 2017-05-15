# Ivy Playground

## Introduction

All value on a Chain blockchain is stored in contracts. Each contract stores some value (i.e., some number of units of a particular asset ID). The value in the contract is protected by a control program.

Ivy is used to define contract templates. To create a contract, you pass arguments to a contract template.

This is an example of a contract template:

```
contract LockWithPublicKey(publicKey: PublicKey, locked: Value) {
  clause unlock(sig: Signature) {
    verify checkTxSig(publicKey, sig)
    return locked  
  }
}
```

This contract, named `LockWithPublicKey` takes two contract parameters: `publicKey` and `locked`. The contract must specify a type for each of those parameters (here, `PublicKey` and `Value`, respectively). Every contract must have a single contract parameter of type `Value`, which must be the last parameter to the contract.

The arguments corresponding to these parameters are passed at the time the contract is created.

### Values

Unlike the other arguments to the contract, the Value doesn’t just represent data. Instead, it represents actual units of a scarce asset. The Value type also behaves like a linear type. Each clause must either return, or output into a contract, all of the items of type Value under its control (in other words, both the contract arguments and the clause arguments of type Value). Each item of type Value can be output or return only once.

### Clauses

Each contract has one or more clauses. To consume a contract (and be able to use the value controlled by it), the caller must provide arguments for each of those clauses. The clause parameters themselves might include Value parameters, which must be either returned or output by that clause.

### Types

Ivy is statically typed. Each parameter specifies a type, and each variable has type statically assigned at compile time. (Variables cannot be reassigned.)

| String | A bytestring. |
|-----------------------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| Number | A 64-bit signed integer. |
| Boolean | Either true or false. |
| PublicKey | An Ed25519 public key. |
| Signature | An Ed25519 signature. |
| Hash | The result of a hash function. Hashes that use different hash functions, or which are hashes of inputs with different types, cannot be compared to each other. |
| Time | A time (specified in milliseconds). |
| AssetID | A 32-byte asset ID. |
| AssetAmount | A pair of a specific AssetID and a specific non-negative Number. |
| Value | Actual units of an asset. To pass an argument of type Value to a contract or a clause, the caller must actually spend that value into the transaction. The AssetAmount of an item of type Value can be inspected like this: someValue.assetAmount. |
| Program | A control program (in other words, a Contract that has not yet been instantiated with a Value). |
| Contract | Some amount of actual value protected by a control program. Cannot be used as the type for contract parameters or clause parameters, but can be created by instantiating a Program with a particular Value, and can then be output to the blockchain (e.g., output someProgram(someValue)). |
| List (such as PublicKey[], Signature[]) | An immutable list of items of the same type. Currently, contract and clause parameters cannot have a List type. The only way to create a list is as a literal: [publicKey1, publicKey2]. |

### Statements

#### verify (expression: Boolean)

Halts execution (causing the contract evaluation to fail) if the expression does not evaluate to true.

```
verify 1 + 1 == 2
verify checkTxSig(publicKey, sig)
verify sha3(str) == stringHash
```

#### return (value: Value)

Returns the specified Value to the caller, who is free to use it as they wish.

```
contract FreeValue(value: Value) {
  clause take() {
    return value
  }
}
```

#### output (contract: Contract)

Outputs a newly created contract to the blockchain.

```
contract SendToAny(value: Value) {
  clause send(program: Program) {
    output program(value)
  }
}
```

### Functions

#### checkTxSig(publicKey: PublicKey, signature: Signature): Boolean

Returns true if the signature is a valid signature on the transaction’s signature hash (the transaction ID concatenated with the input ID) using the corresponding publicKey. Otherwise, returns false.

#### checkMultiSig(publicKeys: PublicKey[], signatures: Signature[]): Boolean

Returns true if each of the signatures corresponds uniquely to one of the publicKeys as a signature on the transaction’s signature hash. The signatures must be in the same order as the public keys to which they correspond.

#### sha256(preimage: String | PublicKey | Signature | Hash): Hash

Returns the SHA-256 hash of preimage.

#### sha3(preimage: String | PublicKey | Signature | Hash): Hash

Returns the SHA3-256 hash of preimage.

#### min(a: Number, b: Number): Number

Returns the lower of a and b.

#### max(a: Number, b: Number): Number

Returns the higher of a and b.

#### abs(a: Number): Number

Returns the absolute value of a.

#### size(s: String): Number

Returns the length of s in bytes.

#### tx.after(t: Time): Boolean

Returns true if the transaction can only be added to the blockchain after time t.

#### tx.before(t: Time): Boolean

Returns true if the transaction can only be added to the blockchain until time t.

### Operators

#### Comparison Operators

Comparison operators compare two operands and return a Boolean value. The operands must have the same type.

The equality operators can only be used on operands of the same type.

== (equal)
!= (not equal)

The ordering operators can only be used on Numbers (otherwise, the script does not compile):

< (less than)
<= (less than or equal)
> (greater than)
> (greater than or equal

#### Arithmetic Operators

The arithmetic operators can only be used on arguments with type Number.

+ (add)
- (subtract)

#### Unary Operators

The not operator ! can only be used on a value of type Boolean.

The negation operator - can only be used on a value of type Number.

### Literals

#### Boolean Literals

true
false

#### Number Literals

0
15
-20

#### List Literals

[pubKey1, pubKey2]
