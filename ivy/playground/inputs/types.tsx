import {
  HashFunction,
  ClauseParameter,
  ClauseParameterType,
  ClauseParameterHash // clause parameters are a superset of contract parameters
} from 'ivy-compiler'

import { Template } from '../templates/types'

export type Input = ParameterInput | StringInput | HashInput | PublicKeyInput | NumberInput |
  BooleanInput | TimeInput | GenerateStringInput | ProvideStringInput | GenerateHashInput |
  ProvideHashInput | GeneratePublicKeyInput | ProvidePublicKeyInput | GeneratePrivateKeyInput |
  ProvidePrivateKeyInput | TimestampTimeInput |
  SignatureInput | GenerateSignatureInput | ProvideSignatureInput | AddressInput | 
  ValueInput | AccountAliasInput | AssetAliasInput | AmountInput |
  AddressInput | AssetAmountInput | ChoosePublicKeyInput

export type ComplexInput = StringInput | HashInput | GenerateHashInput | PublicKeyInput | TimeInput |
  ParameterInput | GeneratePublicKeyInput | SignatureInput | GenerateSignatureInput | AddressInput |
  AddressInput

export type InputType = "parameterInput" | "stringInput" | "generateStringInput" | "provideStringInput" |
  "hashInput" | "generateHashInput" | "provideHashInput" | "publicKeyInput" | "generatePublicKeyInput" |
  "providePublicKeyInput" | "generatePrivateKeyInput" | "providePrivateKeyInput" | "numberInput" | "booleanInput" |
  "timeInput" | "timestampTimeInput" | "signatureInput" | "generateSignatureInput" | "provideSignatureInput" | "addressInput" |
  "valueInput" | "accountAliasInput" |
  "assetAliasInput" | "amountInput" | "addressInput" | "assetAmountInput" | "choosePublicKeyInput"

export type PrimaryInputType = "stringInput" | "hashInput" | "publicKeyInput" | "numberInput" | "booleanInput" |
  "timeInput" | "signatureInput" | "valueInput" | "addressInput" | "assetAmountInput"

export type InputContext = "contractParameters"|"clauseParameters"|"transactionDetails"

export type InputMap = {[s: string]: Input}

export type ParameterInput = {
  type: "parameterInput",
  value: PrimaryInputType,
  valueType: ClauseParameterType,
  name: string
}

export type StringInput = {
  type: "stringInput",
  value: "provideStringInput"|"generateStringInput",
  name: string
}

export type GenerateStringInput = {
  type: "generateStringInput",
  value: string, // length
  seed: string,
  name: string
}

export type ProvideStringInput = {
  type: "provideStringInput",
  value: string,
  name: string
}

export type HashInput = {
  type: "hashInput",
  hashType: ClauseParameterHash,
  value: "provideHashInput"|"generateHashInput",
  name: string
}

export type GenerateHashInput = {
  type: "generateHashInput",
  hashType: ClauseParameterHash,
  value: string,
  name: string
}

export type ProvideHashInput = {
  type: "provideHashInput",
  hashFunction: HashFunction,
  value: string,
  name: string
}

export type KeyMap = {[s: string]: KeyData}

export type KeyData = {
  rootXpub: string,
  pubkeyDerivationPath: string[]
}

export type PublicKeyInput = {
  type: "publicKeyInput",
  value: "accountAliasInput",//"providePublicKeyInput"|"generatePublicKeyInput",
  name: string,
  computedData?: string,
  keyData?: KeyData
}

export type ProvidePublicKeyInput = {
  type: "providePublicKeyInput",
  value: string,
  name: string
}

export type ChoosePublicKeyInput = {
  type: "choosePublicKeyInput",
  value: string,
  name: string,
  keyMap?: KeyMap
}

export type ProvidePrivateKeyInput = {
  type: "providePrivateKeyInput",
  value: string,
  name: string
}

export type GeneratePublicKeyInput = {
  type: "generatePublicKeyInput",
  value: "generatePrivateKeyInput"|"providePrivateKeyInput",
  name: string
}

export type GeneratePrivateKeyInput = {
  type: "generatePrivateKeyInput",
  value: string, // secret
  name: string
}

export type NumberInput = {
  type: "numberInput"
  value: string,
  name: string
}

export type BooleanInput = {
  type: "booleanInput",
  value: "true"|"false",
  name: string
}

export type TimeInput = {
  type: "timeInput",
  value: "blockheightTimeInput"|"timestampTimeInput",
  name: string
}

export type TimestampTimeInput = {
  type: "timestampTimeInput",
  value: string,
  name: string
}

// signatures and addresses are only clause inputs, for now

export type SignatureInput = {
  type: "signatureInput",
  value: "choosePublicKeyInput",//"generateSignatureInput" | "provideSignatureInput"
  name: string
}

export type ProvideSignatureInput = {
  type: "provideSignatureInput",
  value: string,
  name: string
}

export type GenerateSignatureInput = {
  type: "generateSignatureInput",
  value: "providePrivateKeyInput", // (for now this is the only option)
  name: string
}

export type ValueInput = {
  type: "valueInput",
  value: string,
  name: string
}

export type AccountAliasInput = {
  type: "accountAliasInput",
  value: string,
  name: string
}

export type AssetAliasInput = {
  type: "assetAliasInput",
  value: string,
  name: string
}

export type AmountInput = {
  type: "amountInput",
  value: string,
  name: string
}

export type AddressInput = {
  type: "addressInput",
  value: string, // for now just "accountAliasInput"
  name: string
  computedData?: string
}

export type AssetAmountInput = {
  type: "assetAmountInput",
  value: string,
  name: string
}
