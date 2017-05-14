export type Type = Primitive | TypeVariable | Hash | List | "SigHash" | "Contract"

export type Primitive = "PublicKey" | "Signature" | "String" | "Integer" | "Time" | "Boolean" |
                        "Value" | "Asset" | "Amount" | "Program" | "Sha3(PublicKey)" | "Sha3(String)"

export type DeclarableType = Primitive | "Hash"

export type HashFunction = "sha256" | "sha3"

export type Hash = { type: "hashType", hashFunction: HashFunction, inputType: Type }

export type List = { type: "listType", elementType: Type }

export type TypeClass = "Primitive" | "TypeVariable" | "Hash" | "List" | "Other"

export type TypeVariable = { type: "typeVariable", name: string }

export type ContractParameterHash = {
  type: "hashType",
  inputType: ContractParameterType,
  hashFunction: HashFunction
}

export type ClauseParameterHash = {
  type: "hashType",
  inputType: ClauseParameterType,
  hashFunction: HashFunction
}

export type ClauseParameterType = "Signature" | ClauseParameterHash | ContractParameterType

export type ContractParameterType = "PublicKey" | "String" | "Integer" | "Time" | "Boolean" |
                                    "Program" | "Asset" | "Amount" | "Value" | ContractParameterHash
export type ContractParameter = {
  type: "contractParameter",
  valueType: ContractParameterType,
  name: string
}

export type ClauseParameter = {
  type: "clauseParameter",
  valueType: ClauseParameterType,
  name: string
}

export type Input = ParameterInput | StringInput | HashInput | PublicKeyInput | NumberInput | BooleanInput | TimeInput |
                    GenerateStringInput | ProvideStringInput | GenerateHashInput | ProvideHashInput | GeneratePublicKeyInput |
                    ProvidePublicKeyInput | GeneratePrivateKeyInput | ProvidePrivateKeyInput | TimestampTimeInput | SignatureInput |
                    GenerateSignatureInput | ProvideSignatureInput | ProgramInput | ValueInput | AccountAliasInput | AssetAliasInput |
                    AmountInput | ProgramInput | AssetInput | ChoosePublicKeyInput

export type ComplexInput = StringInput | HashInput | GenerateHashInput | PublicKeyInput | TimeInput | ParameterInput | GeneratePublicKeyInput |
                           SignatureInput | GenerateSignatureInput | ProgramInput

export type InputType = "parameterInput" | "stringInput" | "generateStringInput" | "provideStringInput" | "hashInput" |
                        "generateHashInput" | "provideHashInput" | "publicKeyInput" | "generatePublicKeyInput" | "providePublicKeyInput" |
                        "generatePrivateKeyInput" | "providePrivateKeyInput" | "numberInput" | "booleanInput" | "timeInput" |
                        "timestampTimeInput" | "signatureInput" | "generateSignatureInput" | "provideSignatureInput" | "programInput" |
                        "valueInput" | "accountInput" | "assetInput" | "programInput" | "assetInput" | "amountInput" | "choosePublicKeyInput"

export type PrimaryInputType = "stringInput" | "hashInput" | "publicKeyInput" | "numberInput" | "booleanInput" | "timeInput" | "signatureInput" |
                               "valueInput" | "programInput" | "assetInput" | "amountInput"

export type InputContext = "contractParameters"|"clauseParameters"|"clauseValue"|"contractValue"|"unlockValue"

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
  value: "accountInput" | "provideStringInput",
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

// signatures and programs are only clause inputs, for now

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
  type: "accountInput",
  value: string,
  name: string
}

export type AssetAliasInput = {
  type: "assetInput",
  value: string,
  name: string
}

export type AmountInput = {
  type: "amountInput",
  value: string,
  name: string
}

export type ProgramInput = {
  type: "programInput",
  value: string, // for now just "accountInput"
  name: string,
  computedData?: string
}

export type AssetInput = {
  type: "assetInput",
  value: string,
  name: string
}

export function isHash(type: Type): type is Hash {
  return (typeof type === "object" && type.type === "hashType")
}
