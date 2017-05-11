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
  toContractParameter, ContractParameter
} from './cvm/parameters'

import {
  assemble
} from './cvm/assemble'

function compileAst(source: string) {
  let parsed = parser.parse(source)
  let refChecked = referenceCheck(parsed)
  return typeCheckContract(refChecked)
}

export function compileContractParameters(source: string) {
  let ast = compileAst(source)
  return ast.parameters.map(toContractParameter)
}

export function compileTemplateClauses(source: string) {
  let ast = compileAst(source)
  return ast.clauses.map(toTemplateClause)
}

export function compileTemplate(source: string): Template|CompilerError {
  try {
    let ast = compileAst(source)
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

