import {
  TypeSignature,
  createTypeSignature
} from './types'

import {
  BugError
} from '../errors'


export type UnaryOperator = "!"|"negate" // to avoid ambiguity

export function isDeclarableUnaryOperator(str: string): str is "!"|"-" { // this is really for the parser
  return (str === "!") || (str === "-")
}

export type ComparisonOperator = "=="|"!="|">"|"<"|">="|"<="

export function isComparisonOperator(str: string): str is ComparisonOperator {
  return ["==", "!=", ">", "<", ">=", "<="].indexOf(str) !== -1
}

export type ArithmeticOperator = "+"|"-"

export function isArithmeticOperator(str: string): str is ArithmeticOperator {
  return ["+", "-"].indexOf(str) !== -1
}

export type FunctionName = "checkTxSig"|"sha256"|"sha3"|
                           "min"|"max"|"abs"|
                           "size"|"tx.after"|"tx.before"|"tx.sighash"|
                           "checkMultiSig"

export type Opcode = String // for now

export type BinaryOperator = ComparisonOperator | ArithmeticOperator

export type Instruction = UnaryOperator | BinaryOperator | FunctionName

// slightly hackish runtime type guard

export function isInstruction(instructionName: Instruction | string): instructionName is Instruction {
  const opcodes = getOpcodes(instructionName as Instruction)
  return (opcodes !== undefined)
}

export function getOpcodes(instruction: Instruction): Opcode[] {
  switch (instruction) {
    case "checkTxSig":    return ["CHECKSIG"] // special treatment - tx.sigHash is implicit
    case "sha256":        return ["SHA256"]
    case "sha3":          return ["SHA3"]
    case "min":           return ["MIN"]
    case "max":           return ["MAX"]
    case "abs":           return ["ABS"]
    case "size":          return ["SIZE", "SWAP", "DROP"]
    case "tx.before":     return ["MAXTIME", "LESSTHANOREQUAL"]
    case "tx.after":      return ["MINTIME", "GREATERTHANOREQUAL"]
    case "tx.sighash":    return ["TXSIGHASH"]
    case "checkMultiSig": return ["CHECKMULTISIG"] // will get special treatment
    case "negate":        return ["NEGATE"]
    case "!":             return ["NOT"]
    case "+":             return ["ADD"]
    case "-":             return ["SUB"]
    case "==":            return ["EQUAL"]
    case "!=":            return ["EQUAL", "NOT"]
    case "<":             return ["LESSTHAN"]
    case ">":             return ["GREATERTHAN"]
    case ">=":            return ["GREATERTHANOREQUAL"]
    case "<=":            return ["LESSTHANOREQUAL"]
  }
}

export function getTypeSignature(instruction: Instruction): TypeSignature {
  switch (instruction) {
    case "+":
    case "-":
    case "min":
    case "max":
      return createTypeSignature(["Number", "Number"], "Number")
    case "==":
    case "!=":
      return createTypeSignature([{type: "typeVariable", name: "A"}, {type: "typeVariable", name: "A"}], "Boolean")
    case "<":
    case ">":
    case ">=":
    case "<=":
      return createTypeSignature(["Number", "Number"], "Boolean")
    case "!":
      return createTypeSignature(["Boolean"], "Boolean")
    case "negate":
    case "abs":
      return createTypeSignature(["Number"], "Number")
    case "sha256":
    case "sha3":
      return createTypeSignature([{ type: "typeVariable", name: "A"}], 
                                  { type: "hashType", hashFunction: instruction, inputType: { type: "typeVariable", name: "A"} })
    case "checkTxSig":
      return createTypeSignature(["PublicKey", "Signature"], "Boolean")
    case "size":
      return createTypeSignature(["String"], "Number")
    case "tx.after":
    case "tx.before":
      return createTypeSignature(["Time"], "Boolean")
    case "tx.sighash":
      return createTypeSignature([], "String")
    case "checkMultiSig":
      return createTypeSignature([{ type: "listType", elementType: "PublicKey" }, 
                                  { type: "listType", elementType: "Signature" }],
                                 "Boolean")
  }
}
