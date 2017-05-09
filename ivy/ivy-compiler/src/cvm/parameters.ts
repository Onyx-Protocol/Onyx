import {
  Hash,
  HashFunction,
  Type,
  isList,
  isTypeVariable,
  typeToString,
  isHash
} from './types'

import {
  Parameter
} from '../ast'

import {
  BugError
} from '../errors'

export type ContractParameterHash =
  { type: "hashType",
    inputType: ContractParameterType,
    hashFunction: HashFunction
  }

export type ContractParameterType = "PublicKey" | "String" | "Integer" | 
                                    "Time" | "Boolean" | "Address" |
                                    "AssetAmount" | "Value" | ContractParameterHash

export type ContractParameter = {
  type: "contractParameter",
  valueType: ContractParameterType,
  identifier: string
}

export type ClauseParameter = {
  type: "clauseParameter",
  valueType: ClauseParameterType,
  identifier: string
}

export function isContractParameterHash(type: Hash): type is ContractParameterHash {
  return isContractParameterType(type.inputType)
}

export function isContractParameterType(type: Type|"Hash"): type is ContractParameterType {
  if (type === "Hash" || type === "Signature" || isList(type) || isTypeVariable(type)) // "Hash" is a generic hash
    return false
  if (isHash(type)) {
    return isContractParameterHash(type)
  } else {
    return true
  }
}

export function toContractParameter(parameter: Parameter): ContractParameter {
  if (!isContractParameterType(parameter.itemType)) throw new BugError("invalid contract parameter type for " + 
                                                                       parameter.identifier + ": " + 
                                                                       typeToString(parameter.itemType))
  let contractParameter: ContractParameter = {
    type: "contractParameter",
    valueType: parameter.itemType,
    identifier: parameter.identifier
  }
  return contractParameter
}

export type ClauseParameterHash = { 
  type: "hashType",
  inputType: ClauseParameterType,
  hashFunction: HashFunction
}

export type ClauseParameterType = "Signature" | ClauseParameterHash | ContractParameterType

export function isClauseParameterHash(type: Hash): type is ClauseParameterHash {
  return isClauseParameterType(type.inputType)
}

export function isClauseParameterType(type: Type|"Hash"): type is ClauseParameterType {
  if (type === "Hash") return false
  if (isHash(type)) {
    return isClauseParameterHash(type)
  }
  if (type === "Signature") return true
  return isContractParameterType(type)
}

export function toClauseParameter(parameter: Parameter): ClauseParameter {
  if (!isClauseParameterType(parameter.itemType)) throw new BugError("invalid contract parameter type for " + 
                                                                       parameter.identifier + ": " + 
                                                                       typeToString(parameter.itemType))
  let clauseParameter: ClauseParameter = {
    type: "clauseParameter",
    valueType: parameter.itemType,
    identifier: parameter.identifier
  }
  return clauseParameter
}
