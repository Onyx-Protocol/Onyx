import {
  BugError
} from '../errors'

export type Type = Primitive | TypeVariable | Hash | List | "SigHash" | "Contract"

export type Primitive = "PublicKey" | "Signature" | "String" | "Integer" | 
                        "Time" | "Boolean" | "Value" | "AssetAmount" |
                        "Address"

export type DeclarableType = Primitive | "Hash"

export type HashFunction = "sha256" | "sha3"

export type Hash = { type: "hashType", hashFunction: HashFunction, inputType: Type }

export type List = { type: "listType", elementType: Type }

export type TypeClass = "Primitive" | "TypeVariable" | "Hash" | "List" | "Other"

export type TypeVariable = { type: "typeVariable", name: string }

export type TypeSignature = {
  type: "typeSignature",
  inputs: Type[],
  output: Type,
}

export function createTypeSignature(inputs: Type[], output: Type): TypeSignature {
  return {
    type: "typeSignature",
    inputs: inputs,
    output: output,
  }
}

export function inputTypesToString(inputTypes: Type[]) {
  return "(" + inputTypes.map(type => typeToString(type))
                   .join(", ") + ")"
}

export function isPrimitive(str: Type|string): str is Primitive {
  switch (str) {
    case "PublicKey":
    case "Signature":
    case "String":
    case "Integer":
    case "Time":
    case "Boolean":
    case "Value":
    case "AssetAmount":
    case "Address":
      return true
    default:
      return false
  }
}

export function isDeclarableType(str: string): str is DeclarableType {
  return isPrimitive(str) || str == "Hash"
}

export function isHash(type: Type): type is Hash {
  return (typeof type === "object" && type.type === "hashType")
}

export function isList(type: Type): type is List {
  return (typeof type === "object" && type.type === "listType")
}

export function isTypeVariable(type:Type): type is TypeVariable {
  return (typeof type === "object" && (type.type === "typeVariable"))
}

export function isTypeClass(type:Type|TypeClass): type is TypeClass {
  return (type === "Primitive" || type === "TypeVariable" || type === "Hash" || type === "List")
}

export function getTypeClass(type: Type): TypeClass {
  if (isPrimitive(type)) return "Primitive"
  else if (type === "SigHash" || type === "Contract") return "Other"
  else if (isHash(type)) return "Hash"
  else if (isList(type)) return "List"
  else if (isTypeVariable(type)) return "TypeVariable"
  else throw new BugError("unknown typeclass: " + typeToString(type))
}

export function hashFunctionToTypeName(hash: HashFunction): string {
  switch(hash) {
    case "sha256": return "Sha256"
    case "sha3": return "Sha3"
  }
}

export function typeToString(type: Type|TypeClass): string {
  if (isTypeClass(type)) return type
  if (type === undefined) throw new BugError("undefined passed to typeToString()")
  if (typeof type === "object") {
    switch (type.type) {
      case "hashType":
        return hashFunctionToTypeName(type.hashFunction) + "<" + typeToString(type.inputType) + ">"
      case "listType":
        return "List<" + typeToString(type.elementType) + ">"
      case "typeVariable":
        return type.name
      default:
        throw new BugError("unknown type")
    }
  } else {
    return type
  }
}

