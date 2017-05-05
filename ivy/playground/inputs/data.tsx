import { Return } from '../../ivy-compiler/src/ast';
import * as React from 'react'
import { createSelector } from 'reselect'

import {
  isHash,
  HashFunction,
  ClauseParameter,
  ClauseParameterType,
  ClauseParameterHash // clause parameters are a superset of contract parameters
} from 'ivy-compiler'

import {
  Input,
  ComplexInput,
  InputType,
  InputMap,
  PrimaryInputType,
  InputContext,
  HashInput,
  AddressInput,
  PublicKeyInput,
  GenerateStringInput,
  GenerateHashInput,
  ParameterInput,
  ProvideHashInput,
  KeyMap
} from './types'

let Promise = require('prfun')

import {
  crypto
} from 'bcoin'

import {
  client
} from '../util'

import {
  sha3_256
} from 'js-sha3'

import * as app from '../app/types'
import { SpendFromAccount } from '../transactions/types'
import { MIN_TIMESTAMP, MAX_NUMBER, MIN_NUMBER, MAX_UINT32, MAX_UINT16 } from './constants'
import templates from '../templates'

export const computeDataForInput = (id: string, inputMap: InputMap): string => {
  let data = getData(id, inputMap)
  if (typeof data === "number") throw "should not get data for a number"
  return data.toString('hex')
}

export function getData(inputId: string, inputsById: {[s: string]: Input}): Buffer|number {
  let input = inputsById[inputId]
  if (!validateInput(input)) throw "invalid input: " + input.name
  switch (input.type) {
    case "parameterInput": 
    case "stringInput":
    case "hashInput":
    case "durationInput":
    case "mintimeInput":
    case "maxtimeInput":
    case "timeInput":
    case "signatureInput":
    case "durationInput":
      return getData(getChild(input), inputsById)
    case "addressInput":
    case "publicKeyInput": {
      if (input.computedData === undefined) throw "input.computedData unexpectedly undefined"
      return Buffer.from(input.computedData, 'hex')
    }
    case "provideStringInput":
    case "providePublicKeyInput":
    case "provideHashInput":
    case "provideSignatureInput":
    case "assetAliasInput": {
      return Buffer.from(input.value, 'hex')
    }
    // case "generatePublicKeyInput": {
    //   let publicKeyValue = getPublicKeyValue(inputId, inputsById)
    //   return Buffer.from(publicKeyValue, "hex")
    // }
    case "numberInput":
    case "amountInput": {
      return parseInt(input.value, 10)
    }
    case "booleanInput": {
      return (input.value === "true") ? 0 : 1
    }
    case "blocksDurationInput":
    case "secondsDurationInput": {
      let numValue = parseInt(input.value, 10)
      let buf = Buffer.alloc(4)
      buf.writeUInt16LE(numValue, 2)
      if (input.type === "secondsDurationInput") { buf.writeUInt8(64, 1) } // set the flag
      return buf.readUInt32LE(0)
    }
    case "timestampTimeInput":
    case "blockheightTimeInput": {
      let numValue = parseInt(input.value, 10)
      let buf = Buffer.alloc(4)
      buf.writeUInt32LE(numValue, 0)
      return buf.readUInt32LE(0)
    }
    case "generateStringInput": {
      let generated = getGenerateStringInputValue(input)
      return Buffer.from(generated, 'hex')
    }
    case "generateHashInput": {
      let childData = getData(getChild(input), inputsById)
      if (typeof childData === "number") throw "should not generate hash of a number"
      switch(input.hashType.hashFunction) {
        case "sha256": return crypto.sha256(childData)
        case "sha3": return Buffer.from(sha3_256(childData), "hex")
        default: throw "unexpected hash function"
      }
    }
    // case "generateSignatureInput": {
    //   throw " "
    //   let privKey = getPrivateKeyValue(inputId, inputsById)
    //   if (sigHash === undefined) throw "no sigHash provided to generateSignatureInput"
    //   return createSignature(sigHash, privKey, inputsById)
    // }
    default:
      throw "should not call getData with " + input.type
  }
}

export const getInputNameContext = (name: string) => {
  return name.split(".")[0] as InputContext
}

export const getInputContext = (input: Input): InputContext => {
  return getInputNameContext(input.name)
}

export const getParameterIdentifier = (input: ParameterInput): string => {
  switch (getInputContext(input)) {
    case "contractParameters": return input.name.split(".")[1]
    case "clauseParameters": return input.name.split(".")[2]
    default:
      throw "unexpected input for getParameterIdentifier: " + input.name
  }
}

export const getChild = (input: ComplexInput): string => {
  return input.name + "." + input.value
}

export const isPrimaryInputType = (str: string): str is PrimaryInputType => {
  switch (str) {
    case "hashInput":
    case "numberInput":
    case "booleanInput":
    case "stringInput":
    case "publicKeyInput":
    case "durationInput":
    case "timeInput":
    case "signatureInput":
    case "valueInput":
    case "addressInput":
    case "assetAmountInput":
      return true
    default:
      return false
  }
}

export const isComplexType = (inputType: InputType) => {
  switch (inputType) {
    case "parameterInput":
    case "generateHashInput":
    case "hashInput":
    case "stringInput":
    case "publicKeyInput":
    case "durationInput":
    case "timeInput":
    case "generatePublicKeyInput":
    case "generateSignatureInput":
    case "signatureInput":
    case "addressInput":
    case "mintimeInput":
    case "maxtimeInput":
    case "addressInput":
      return true
    default:
      return false
  }
}

export const isComplexInput = (input: Input): input is ComplexInput => {
  return isComplexType(input.type)
}

export const getInputType = (type: ClauseParameterType): PrimaryInputType => {
  if (isHash(type)) return "hashInput"
  switch (type) {
    case "Number": return "numberInput"
    case "Boolean": return "booleanInput"
    case "String": return "stringInput"
    case "PublicKey": return "publicKeyInput"
    case "Time": return "timeInput"
    case "Signature": return "signatureInput"
    case "Value": return "valueInput"
    case "Address": return "addressInput"
    case "AssetAmount": return "assetAmountInput"
    default:
      throw "can't yet get input type for " + type
  }
}

export const isValidInput = (id: string, inputMap: InputMap): boolean => {
  const input = inputMap[id]
  switch (input.type) {
    case "parameterInput":
    case "stringInput":
    case "hashInput":
    case "publicKeyInput":
    case "durationInput":
    case "mintimeInput":
    case "maxtimeInput":
    case "timeInput":
    case "signatureInput":
    case "durationInput":
    case "addressInput":
    case "timeInput":
    case "signatureInput":
      return isValidInput(getChild(input), inputMap)
    case "assetAmountInput":
      return isValidInput(input.name + ".assetAliasInput", inputMap) &&
             isValidInput(input.name + ".amountInput", inputMap)
    case "valueInput":
      return isValidInput(input.name + ".accountAliasInput", inputMap) &&
             isValidInput(input.name + ".assetAmountInput", inputMap)
    default: return validateInput(input)
  }
}

const validateHex = (str: string): boolean => {
  return (/^([a-f0-9][a-f0-9])*$/.test(str))
}

export const validateInput = (input: Input): boolean => {
  // validates that an individual
  // does not validate child inputs
  let numberValue
  switch (input.type) {
    case "parameterInput":
    case "generateHashInput":
      return isPrimaryInputType(input.value)
    case "stringInput":
      return (input.value === "generateStringInput" ||
              input.value === "provideStringInput")
    case "hashInput":
      return (input.value === "generateHashInput" ||
              input.value === "provideHashInput")
    case "publicKeyInput":
      return (input.value === "accountAliasInput")
    case "generatePublicKeyInput":
      return (input.value === "generatePrivateKeyInput" ||
              input.value === "providePrivateKeyInput")
    case "durationInput":
      return (input.value === "secondsDurationInput" ||
              input.value === "blocksDurationInput")
    case "timeInput":
      return (input.value === "timestampTimeInput")
    case "generatePublicKeyInput":
    case "signatureInput":
      return input.value === "choosePublicKeyInput"
    case "generateSignatureInput":
      return (input.value === "providePrivateKeyInput")
    case "addressInput":
      return (input.value === "accountAliasInput")
    case "provideStringInput":
      return validateHex(input.value)
    case "provideHashInput":
      if (!validateHex(input.value)) return false
      switch (input.hashFunction) {
        case "sha256":
        case "sha3":
          return input.value.length === 64
        default:
          throw 'unsupported hash function: ' + input.hashFunction
      }
    case "generateStringInput": {
      let length = parseInt(input.value, 10)
      if (isNaN(length) || length < 0 || length > 520) return false
      return (input.seed.length == 1040)
    }
    case "booleanInput":
      return (input.value === "true" ||
              input.value === "false")
    case "numberInput":
    case "timestampTimeInput":
    case "blockheightTimeInput":
    case "secondsDurationInput":
    case "blocksDurationInput":
      numberValue = parseInt(input.value, 10)
      if (isNaN(numberValue)) return false
      switch(input.type) {
        case "numberInput":
          return (numberValue >= MIN_NUMBER && 
                  numberValue <= MAX_NUMBER)
        case "timestampTimeInput":
          return (numberValue >= MIN_TIMESTAMP &&
                  numberValue <= MAX_UINT32)
        case "blockheightTimeInput":
          return (numberValue >= 0 &&
                  numberValue < MIN_TIMESTAMP)
        case "secondsDurationInput":
        case "blocksDurationInput":
          return (numberValue >= 0 && 
                  numberValue <= MAX_UINT16)
        default:
          throw "unexpectedly reached end of switch statement"
      }
    case "mintimeInput":
    case "maxtimeInput":
      return input.value === "timeInput"
    case "amountInput":
      numberValue = parseInt(input.value, 10)
      if (isNaN(numberValue)) return false
      if (numberValue < 0) return false
      return true
    case "accountAliasInput":
    case "assetAliasInput":
      return (input.value !== "")
    case "assetAmountInput":
    case "valueInput":
      // TODO(dan)
      return true
    case "choosePublicKeyInput":
      return (input.keyMap !== undefined) && (input.keyMap[input.value] !== undefined)
    default:
      throw 'input type not valid ' + input.type
  }
}

// export const getPublicKeyValue = (inputId: string, inputsById: {[s: string]: Input}) => {
//   let privateKeyValue = getPrivateKeyValue(inputId, inputsById)
//   let kr = keyring.fromSecret(privateKeyValue)
//   return kr.getPublicKey("hex")
// }

export const getGenerateStringInputValue = (input: GenerateStringInput) => {
  let length = parseInt(input.value, 10)
  if (isNaN(length) || length < 0 || length > 520) 
    throw "invalid length value for generateStringInput: " + input.value
  return input.seed.slice(0, length * 2) // dumb, for now
}

function addHashInputs(inputs: Input[], type: ClauseParameterHash, parentName: string) {
  let name = parentName + ".generateHashInput"
  let value = getInputType(type.inputType)
  let generateHashInput: GenerateHashInput = {
    type: "generateHashInput",
    hashType: type,
    value: value,
    name: name
  }
  inputs.push(generateHashInput)
  addInputForType(inputs, type.inputType, name)

  let hashType = generateHashInput.hashType.inputType

  let provideHashInput: ProvideHashInput = {
    type: "provideHashInput",
    hashFunction: type.hashFunction,
    value: "",
    name: parentName + ".provideHashInput"
  }
  inputs.push(provideHashInput)
}

function addHashInput(inputs: Input[], type: ClauseParameterHash, parentName: string) {
  let name = parentName + ".hashInput"
  let hashInput: HashInput = {
    type: "hashInput",
    hashType: type,
    value: "generateHashInput",
    name: name
  }
  inputs.push(hashInput)
  addHashInputs(inputs, type, name)
}



export function getDefaultContractParameterValue(inputType: InputType): string {
  switch (inputType) {
    case "parameterInput":
    case "generateHashInput":
    case "mintimeInput":
    case "maxtimeInput":
      throw "getDefaultContractParameterValue should not be called on " + inputType
    case "booleanInput": 
      return "false"
    case "generateStringInput":
      return "32"
    case "numberInput": 
    case "blocksDurationInput":
    case "secondsDurationInput":
    case "timestampTimeInput":
    case "blockheightTimeInput":
      return "0"
    case "provideStringInput":
    case "provideHashInput":
    case "providePublicKeyInput":
    case "providePrivateKeyInput":
    case "provideSignatureInput":
      return ""
    case "stringInput":
      return "generateStringInput"
    case "hashInput": 
      return "generateHashInput"
    case "generatePublicKeyInput":
      return "generatePrivateKeyInput"
    case "publicKeyInput":
      return "accountAliasInput"
      // return "generatePublicKeyInput"
    case "signatureInput":
      return "choosePublicKeyInput"
      // return "generateSignatureInput"
    case "generateSignatureInput":
      return "providePrivateKeyInput"
    case "addressInput":
      return "accountAliasInput"    
    case "booleanInput":
      return "false"
    case "durationInput":
      return "blocksDurationInput"
    case "timeInput":
      return "timestampTimeInput"
    case "accountAliasInput":
    case "assetAliasInput":
      return ""
    case "valueInput":
    case "assetAmountInput":
      return "" // TODO(dan)
    case "amountInput":
      return ""
    case "choosePublicKeyInput":
    case "generatePrivateKeyInput":
      throw inputType + ' should not be allowed'
  }
}

export function getDefaultTransactionDetailValue(inputType: InputType): string {
  switch (inputType) {
    case "mintimeInput":
    case "maxtimeInput":
      return "timeInput"
    case "addressInput":
      return "generateAddressInput"
    default: // fall back for now
      return getDefaultContractParameterValue(inputType)
  }
}

export function getDefaultClauseParameterValue(inputType: InputType): string {
  switch (inputType) {
    case "parameterInput":
    case "generateHashInput":
    case "mintimeInput":
    case "maxtimeInput":
      throw "getDefaultClauseParameterValue should not be called on " + inputType
    case "booleanInput": 
      return "false"
    case "generateStringInput":
      return "32"
    case "numberInput": 
    case "blocksDurationInput":
    case "secondsDurationInput":
    case "timestampTimeInput":
    case "blockheightTimeInput":
      return "0"
    case "provideStringInput":
    case "provideHashInput":
    case "providePublicKeyInput":
    case "providePrivateKeyInput":
    case "provideSignatureInput":
      return ""
    case "stringInput":
      return "provideStringInput"
    case "hashInput":
      return "provideHashInput"
    case "publicKeyInput":
      return "accountAliasInput"
      // return "providePublicKeyInput"
    case "signatureInput":
      return "choosePublicKeyInput"
      // return "generateSignatureInput"
    case "generatePublicKeyInput":
    case "generateSignatureInput":
      return "providePrivateKeyInput"
    case "booleanInput":
      return "false"
    case "durationInput":
      return "blocksDurationInput"
    case "timeInput":
      return "blockheightTimeInput"
    case "addressInput":
      return "accountAliasInput"
    case "accountAliasInput":
    case "assetAliasInput":
    case "valueInput":
    case "assetAmountInput":
    case "amountInput":
    case "choosePublicKeyInput":
      return "" // TODO?: dan
    case "generatePrivateKeyInput":
      throw inputType + " should not be allowed"
  }
}


export function getPromisedInputMap(inputsById: {[s: string]: Input}): Promise<{[s: string]: Input}> {
  let newInputsById = {}
  for (let id in inputsById) {
    let input = inputsById[id]
    if (input.type === "publicKeyInput" || input.type === "addressInput") {
      newInputsById[id] = getPromiseData(id, inputsById)
    } else {
      newInputsById[id] = input
    }
  }
  return Promise.props(newInputsById)
}

export function getPromiseData(inputId: string, inputsById: {[s: string]: Input}): Promise<Input> {
  let input = inputsById[inputId]
  switch (input.type) {
    case "addressInput": {
      console.log("input", input)
      let accountId = inputsById[input.name + ".accountAliasInput"].value
      return client.accounts.createReceiver({ accountId }).then((receiver) => {
        let addressInput: AddressInput = {
          ...input as AddressInput,
          computedData: receiver.controlProgram
        }
        return addressInput
      })
    }
    case "publicKeyInput": {
      let accountId = inputsById[input.name + ".accountAliasInput"].value
      return client.accounts.createPubkey({ accountId }).then((publicKey) => {
        let publicKeyInput: PublicKeyInput = {
          ...input as PublicKeyInput,
          computedData: publicKey.pubkey,
          keyData: {
            rootXpub: publicKey.rootXpub,
            pubkeyDerivationPath: publicKey.pubkeyDerivationPath
          }
        }
        return publicKeyInput
      })
    }
    default: throw "cannot call getPromiseData with " + input.type
  }
}

export function getDefaultValue(inputType, name): string {
  switch (getInputNameContext(name)) {
    case "clauseParameters": return getDefaultClauseParameterValue(inputType)
    case "contractParameters": return getDefaultContractParameterValue(inputType)
    case "transactionDetails": return getDefaultTransactionDetailValue(inputType)
  }
}

export function addDefaultInput(inputs: Input[], inputType: InputType, parentName) {
  let name = parentName + "." + inputType
  let value = getDefaultValue(inputType, name)
  switch (inputType) {
    case "generateStringInput": {
      let seed = crypto.randomBytes(520).toString('hex')
      inputs.push({
        type: "generateStringInput",
        value: value as any,
        seed: seed,
        name: name
      })
      break
    }
    default:
      inputs.push({
        type: inputType as any,
        value: value,
        name: name
      })
  }
  switch (inputType) {
    case "stringInput": {
      addDefaultInput(inputs, "generateStringInput", name)
      addDefaultInput(inputs, "provideStringInput", name)
      return
    }
    case "publicKeyInput": {
      addDefaultInput(inputs, "accountAliasInput", name)
      // addDefaultInput(inputs, "generatePublicKeyInput", name)
      // addDefaultInput(inputs, "providePublicKeyInput", name)
      return
    }
    case "generatePublicKeyInput": {
      addDefaultInput(inputs, "generatePrivateKeyInput", name)
      addDefaultInput(inputs, "providePrivateKeyInput", name)
    }
    case "timeInput": {
      addDefaultInput(inputs, "timestampTimeInput", name)
      return
    }
    case "durationInput": {
      addDefaultInput(inputs, "blocksDurationInput", name)
      addDefaultInput(inputs, "secondsDurationInput", name)
      return
    }
    case "signatureInput": {
      addDefaultInput(inputs, "choosePublicKeyInput", name)
      // addDefaultInput(inputs, "generateSignatureInput", name)
      // addDefaultInput(inputs, "provideSignatureInput", name)
      return
    }
    case "generateSignatureInput": {
      addDefaultInput(inputs, "providePrivateKeyInput", name)
      return
    }
    case "mintimeInput":
    case "maxtimeInput": {
      addDefaultInput(inputs, "timeInput", name)
      return
    }
    case "valueInput": {
      addDefaultInput(inputs, "accountAliasInput", name)
      addDefaultInput(inputs, "assetAmountInput", name)
      return
    }
    case "assetAmountInput": {
      addDefaultInput(inputs, "assetAliasInput", name)
      addDefaultInput(inputs, "amountInput", name)
      return
    }
    case "addressInput": {
      addDefaultInput(inputs, "accountAliasInput", name)
      return
    }
    default:
      return
  }
}

function addInputForType(inputs: Input[], parameterType: ClauseParameterType, parentName: string) {
  if (isHash(parameterType)) {
    addHashInput(inputs, parameterType, parentName)
  } else {
    addDefaultInput(inputs, getInputType(parameterType), parentName)
  }
}

export function addParameterInput(inputs: Input[], valueType: ClauseParameterType, name: string) {
  let inputType = getInputType(valueType)
  let parameterInput: ParameterInput = {
    type: "parameterInput",
    value: inputType,
    valueType: valueType,
    name: name
  }
  inputs.push(parameterInput)
  addInputForType(inputs, valueType, name)
}

export function getPublicKeys(inputsById: {[s: string]: Input}) {
  let mapping : KeyMap = {}
  for (let id in inputsById) {
    let input = inputsById[id]
    if (input.type === "publicKeyInput") {
      if (input.computedData === undefined) throw 'input.computedData unexpectedly undefined'
      if (input.keyData === undefined) throw 'input.keyData unexpectedly undefined'
      mapping[input.computedData] = input.keyData
    }
  }
  return mapping
}

