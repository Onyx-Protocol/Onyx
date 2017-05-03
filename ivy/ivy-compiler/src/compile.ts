let parser = require('./parser')

import { 
  RawContract,
  Contract,
  contractToString
} from './ast'

import { 
  desugarContract
} from './desugar'

import { 
  typeCheckContract 
} from './typeCheck'

import {
  FinalOperation,
  compileContractToIntermediate,
  operationsToString
} from './intermediate'

import {
  compileStackOps
} from './stack'

import toOpcodes from './cvm/toOpcodes'

import {
  optimize
} from './cvm/optimize'

import {
  BugError,
  NameError,
  IvyTypeError,
  ValueError,
} from './errors'

import {
  Scope
} from './typeCheck'

import {
  referenceCheck
} from './references'

import {
  Template, toTemplateClause, CompilerError
} from './template'

import {
  toContractParameter
} from './cvm/parameters'

import {
  assemble
} from './cvm/assemble'

export function compileTemplate(source: string): Template|CompilerError {
  try {
    let parsed = parser.parse(source)
    // console.log("parsed: ", parsed)
    let refChecked = referenceCheck(parsed)
    // console.log("refChecked: ", refChecked)
    let ast = typeCheckContract(refChecked)
    // console.log("ast: ", ast)
    let templateClauses = ast.clauses.map(toTemplateClause)
    let desugared = desugarContract(ast)
    // console.log("desugared: ", desugared)
    let intermediate = compileContractToIntermediate(desugared)
    // console.log("intermediate: ", intermediate)
    let operations = compileStackOps(intermediate)
    // console.log("operations: ", operations)
    let opcodes = toOpcodes(operations)
    // console.log("opcodes: ", opcodes)
    let instructions = optimize(opcodes)
    // console.log("instructions: ", instructions)
    let contractParameters = ast.parameters.map(toContractParameter)
    return {
      type: "template",
      source: source,
      name: ast.name,
      clauses: templateClauses,
      instructions,
      contractParameters
    }
  } catch (e) {
    // catch and return CompilerError
    let errorMessage: string
    if (e.constructor.name == "peg$SyntaxError" || e.name == "IvyTypeError" ||
                e.name == "NameError" || e.name == "ValueError") {
      if (e.location !== undefined) {
        const start = e.location.start
        const name = (e.name === "IvyTypeError") ? "TypeError" : e.name
        errorMessage = name + " at line " + start.line + ", column " + start.column + ": " + e.message
      } else {
        errorMessage = e.toString()
      }
      return {
        type: "compilerError",
        source: source,
        message: errorMessage
      }
    } else {
      throw e
    }
  }
}

