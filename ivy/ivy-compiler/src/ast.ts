import {
  Primitive,
  typeToString,
  TypeClass,
  Hash
} from './cvm/types'

import {
  Instruction,
  isInstruction
} from './cvm/instructions'

import {
  NameError,
  BugError
} from './errors'

export interface Location { start: { column: number, line: number}, end: { column: number, line: number } }

export type InstructionExpressionType = "unaryExpression"|"binaryExpression"|"callExpression"

export type Parameter = {
  type: "parameter",
  location: Location,
  identifier: string
  itemType: Primitive | Hash | "Hash"
  scope?: string
}

export type RawContract = {
  type: "rawContract",
  location: Location,
  name: string,
  parameters: Parameter[],
  clauses: Clause[],
  referenceCounts?: Map<string, number>
}

export type Conditional = {
  type: "conditional",
  condition: Expression,
  ifBlock: Block,
  elseBlock?: Block,
  endTag: string,
  elseTag?: string
}

export type Block = Conditional | Clause

export type Contract = {
  type: "contract",
  location: Location,
  name: string,
  parameters: Parameter[],
  block: Block,
  numClauses: number,
  clauseSelector?: string
}

export type Clause = {
  type: "clause",
  location: Location,
  name: string,
  parameters: Parameter[],
  assertions: Assertion[],
  outputs: Output[],
  returnStatement?: Return,
  referenceCounts?: Map<string, number>
}

export type StoredValue = { // representing the value stored in the contract
  type: "storedValue",
  location: Location,
  identifier: string
}

export type ContractExpression = {
  type: "contractExpression",
  location: Location,
  program: Variable,
  value: Variable | StoredValue
}

export type Assertion = {
  type: "assertion",
  location: Location,
  expression: Expression
}

export type Return = {
  type: "returnStatement",
  location: Location,
  value: Variable | StoredValue
}

export type Output = {
  type: "output",
  location: Location,
  contract: ContractExpression,
  asset?: string,
  amount?: string,
  assetAmountParam?: string,
  index?: number
}

export type Statement = Assertion | Return | Output

export function statementToString(statement: Statement) {
  switch(statement.type) {
    case "assertion": return "verify " + expressionToString(statement.expression)
    case "returnStatement": return "return" + expressionToString(statement.value)
    case "output": return "output" + expressionToString(statement.contract)
  }
}

export type Expression = InstructionExpression | ListLiteral | ValueLiteral | Variable | ContractExpression | StoredValue

export type InstructionExpression = {
  type: "instructionExpression",
  expressionType: InstructionExpressionType
  instruction: Instruction,
  location: Location,
  args: Expression[]
}

export type PartialExpression = { // this is only for createBinaryExpression
  type: "partial",
  operator: string,
  left: Expression
}

export function createBinaryExpression(partials: PartialExpression[], right: Expression): Expression {
  const last = partials.pop()
  if (last === undefined) throw new BugError("partials list must not be empty")
  const operator = last.operator
  const left = partials.length ? createBinaryExpression(partials, last.left) : last.left
  return createInstructionExpression("binaryExpression", left.location, operator, [right, left])
}

export function createInstructionExpression(expressionType: InstructionExpressionType, location: Location, name: string, args: Expression[]): Expression {
  const instruction = (expressionType === "unaryExpression" && name == "-") ? "negate" : name
  if (!isInstruction(instruction)) {
    throw new NameError("invalid instruction name: " + instruction)
  }

  return {
    type: "instructionExpression",
    expressionType: expressionType,
    instruction: instruction,
    location: location,
    args: args
  }
}

export type Variable = { 
  type: "variable",
  location: Location, 
  identifier: string
  scope?: string
}

export type LiteralType = "Integer" | "Boolean"

export type ListLiteral = {
  type: "listLiteral",
  location: Location,
  values: Expression[]
}

export type ValueLiteral = {
  type: "literal",
  literalType: LiteralType,
  location: Location,
  value: string
}

export function contractToString(contract: RawContract) {
  return "contract " + contract.name + "(" + contract.parameters.map(param => parameterToString(param)).join(", ") +
         ") {\n  " + contract.clauses.map(clause => clauseToString(clause)).join("\n  ") + "\n}"
}

function clauseToString(clause: Clause) {
  return "clause " + clause.name + "(" + clause.parameters.map(param => parameterToString(param)).join(", ") + ") {\n    " +
        clause.assertions.map(assertion => statementToString(assertion)).join("\n    ") + "\n" + 
        clause.outputs.map(output => statementToString(output)).join("\n    ") + "\n" +
        (clause.returnStatement ? statementToString(clause.returnStatement) : "") + "\n   }"
}

function literalToString(literal: ValueLiteral) {
  switch (literal.literalType) {
    case "Integer":
    case "Boolean":
      return literal.value
  }
}

function instructionExpressionToString(expression: InstructionExpression) {
  switch(expression.expressionType) {
    case "unaryExpression":
      if (expression.instruction === "negate") {
        return "-" + expressionToString(expression.args[0])
      } else {
        return expression.instruction + expressionToString(expression.args[0])
      }
    case "binaryExpression":
      return "(" + expressionToString(expression.args[0]) + " " + expression.instruction + " " +
             expressionToString(expression.args[1]) + ")"
    case "callExpression":
      return expression.instruction + "(" + expression.args.map(exp => expressionToString(exp)).join(", ") + ")"
  }
}

function listLiteralToString(expression: ListLiteral) {
  return "[" + expression.values.map(exp => expressionToString(exp)).join(", ") + "]"
}

function expressionToString(expression: Expression): string {
  switch (expression.type) {
    case "literal":
      return literalToString(expression)
    case "instructionExpression":
      return instructionExpressionToString(expression)
    case "variable":
      return scopedName(expression)
    case "listLiteral":
      return listLiteralToString(expression)
    case "contractExpression":
      return expression.program.identifier + "(" + expression.value + ")"
    case "storedValue":
      return expression.identifier
  }
}

function parameterToString(parameter: Parameter) {
  return parameter.identifier + ((parameter.itemType === undefined) ? "" : (": " + typeToString(parameter.itemType)))
}

export type ASTNode = Parameter | RawContract | Contract | Conditional | Clause | Statement | Expression | StoredValue

export function mapOverAST(func: (Node)=>(ASTNode), node: ASTNode): ASTNode {
  switch (node.type) {
    case "parameter": {
      return func(node)
    }
    case "rawContract": {
      return func({
        ...node,
        parameters: node.parameters.map(param => mapOverAST(func, param)),
        clauses: node.clauses.map(clause => mapOverAST(func, clause)),
      })
    }
    case "contract": {
      return func({
        ...node,
        parameters: node.parameters.map(param => mapOverAST(func, param)),
        block: mapOverAST(func, node.block),
      })
    }
    case "conditional": {
      return func({
        ...node,
        condition: mapOverAST(func, node.condition),
        ifBlock: mapOverAST(func, node.ifBlock),
        elseBlock: node.elseBlock ? mapOverAST(func, node.elseBlock) : undefined
      })
    }
    case "clause": {
      return func({
        ...node,
        parameters: node.parameters.map(param => mapOverAST(func, param)),
        assertions: node.assertions.map(st => mapOverAST(func, st)),
        outputs: node.outputs.map(st => mapOverAST(func, st)),
        returnStatement: node.returnStatement ? mapOverAST(func, node.returnStatement) : undefined,
      })
    }
    case "assertion": {
      return func({
        ...node,
        expression: mapOverAST(func, node.expression)
      })
    }
    case "returnStatement": {
      return func({
        ...node,
        value: mapOverAST(func, node.value)
      })
    }
    case "output": {
      return func({
        ...node,
        contract: mapOverAST(func, node.contract)
      })
    }
    case "contractExpression": {
      return func({
        ...node,
        program: mapOverAST(func, node.program),
        value: mapOverAST(func, node.value)
      })
    }
    case "instructionExpression": {
      return func({
        ...node,
        args: node.args.map(arg => mapOverAST(func, arg))
      })
    }
    case "variable": {
      return func(node)
    }
    case "listLiteral": {
      return func({
        ...node,
        values: node.values.map(val => mapOverAST(func, val))
      })
    }
    case "literal": {
      return func(node)
    }
    case "storedValue": {
      return func(node)
    }
  }
}

export function scopedName(item: Parameter | Variable): string {
  return (item.scope === undefined) ? item.identifier : (item.scope + "." + item.identifier)
}
