// compile to an intermediate representation

import {
  Conditional,
  Contract,
  Parameter,
  Clause,
  mapOverAST,
  ASTNode,
  Block,
  Variable,
  InstructionExpression,
  Expression
} from './ast'

import {
  BugError
} from './errors'

import {
  Instruction
} from './cvm/instructions'

export type Operation = Get | Pick | Roll | BeginContract | InstructionOp | Push | 
                        BeginIf | Else | EndIf | BeginClause | EndClause | PushParameter | Drop |
                        Op


export type FinalOperation = Pick | Roll | InstructionOp | Op | Push | BeginIf | Else | EndIf | PushParameter | Drop

export type Get = {
  type: "get",
  variable: Variable
}

export type Pick = {
  type: "pick",
  depth: number
}

export type Roll = {
  type: "roll",
  depth: number
}

export type Drop = {
  type: "drop"
}

export type BeginContract = {
  type: "beginContract",
  contract: Contract
}

export type InstructionOp = {
  type: "instructionOp",
  expression: InstructionExpression
}

export type Push = {
  type: "push",
  literalType: "Number" | "Boolean" | "String",
  value: string
}

export type PushParameter = {
  type: "pushParameter",
  identifier: string
}

export type BeginIf = {
  type: "beginIf",
  elseTag?: string,
  endTag: string
}

export type EndIf = {
  type: "endIf",
  endTag: string
}

export type Else = {
  type: "else",
  elseTag: string,
  endTag: string
}

export type BeginClause = {
  type: "beginClause",
  clause: Clause
}

export type EndClause = {
  type: "endClause",
  clause: Clause
}

export type Op = {
  type: "op",
  name: string,
  numArgs: number,
  numResults: number
}

export function operationToString(op: Operation): string {
  switch (op.type) {
    case "beginIf":
    case "else":
    case "endIf":
      return op.type
    case "op":
      return op.name
    case "beginClause":
      return "(beginClause " + op.clause.name + ")"
    case "endClause":
      return "(endClause " + op.clause.name + ")"
    case "push":
      return (op.literalType === "String") ? "(push 0x" + op.value + ")" : "(push " + op.value + ")"
    case "instructionOp":
      return op.expression.instruction
    case "beginContract":
      return "(beginContract (" + op.contract.parameters.map(param => param.identifier).join(", ") + "))"
    case "get":
      return "(get " + op.variable.identifier + ")"
    case "pick":
      return "(pick " + op.depth + ")"
    case "roll":
      return "(roll " + op.depth + ")"
    case "pushParameter":
      return "(push " + op.identifier + ")"
    case "drop":
      return "drop"
  }
}

export function operationsToString(ops: Operation[]): string {
  return ops.map(operationToString).join(" ")
}

export function compileContractToIntermediate(contract: Contract): Operation[] {
  let operations: Operation[] = []

  let emit = (op: Operation) => (operations.push(op))
  
  compileToIntermediate(contract, emit)

  return operations
}

function compileToIntermediate(node: ASTNode, emit: (op: Operation)=>void): ASTNode {

  let compile = (n => compileToIntermediate(n, emit))

  switch (node.type) {
    case "contract": {
      emit({ type: "beginContract", contract: node })
      compile(node.block)
      return node
    }
    case "rawContract": {
      throw new BugError("raw contract passed to compileToIntermediate, which expects a desugared contract")
    }
    case "clause": {
      emit({ type: "beginClause", clause: node })
      let assertions = node.assertions
      let outputs = node.outputs
      if (assertions.length > 0 || outputs.length > 0) {
        if (outputs.length === 0) {
          assertions.slice(0, -1).map(compile)
          // just the expression from the last statement in each clause
          // don't verify it (because of the implicit verify at the end)
          let expression = assertions[assertions.length - 1].expression
          compile(expression)
        } else {
          assertions.map(compile)
          for (let i = 0; i < outputs.length - 1; i++) {
            compile(outputs[i])
            emit({ type: "op", name: "VERIFY", numArgs: 1, numResults: 0 })
          }
          compile(outputs[outputs.length - 1]) // don't add a VERIFY
        }
      } else {
        // no assertions or outputs, so just add an OP_TRUE
        emit({
          type: "push",
          literalType: "Boolean",
          value: "true"
        })
      }
      emit({ type: "endClause", clause: node })
      return node
    }
    case "conditional": {
      compile(node.condition)
      emit({
        type: "beginIf",
        elseTag: node.elseTag,
        endTag: node.endTag
      })
      compile(node.ifBlock)
      if (node.elseBlock) {
        if (node.elseTag === undefined) throw new BugError("should not have else block without else tag")
        emit({
          type: "else",
          elseTag: node.elseTag,
          endTag: node.endTag
        })
        compile(node.elseBlock)
      }
      emit({
        type: "endIf",
        endTag: node.endTag
      })
      return node
    }
    case "assertion": {
      compile(node.expression)
      emit({ type: "op", name: "VERIFY", numArgs: 1, numResults: 0 })
      return node
    }
    case "output": {
      if (node.index === undefined) {
        throw new BugError("unannotated output node")
      }
      emit({ type: "push", literalType: "Number", value: node.index.toString() }) // index
      emit({ type: "push", literalType: "Number", value: "0" }) // TODO(bobg): explicitly the empty string
      switch (node.contract.value.type) {
        case "variable":
          if (node.assetAmountParam === undefined) {
            throw new BugError("unannotated output variable node")
          }
          emit({
            type: "get",
            variable: {
              ...node.contract.value,
              identifier: node.assetAmountParam + ".amount"
            }
          })
          emit({
            type: "get",
            variable: {
              ...node.contract.value,
              identifier: node.assetAmountParam + ".asset"
            }
          })
          break

        case "storedValue":
          emit({ type: "op", name: "AMOUNT", numArgs: 0, numResults: 1 })
          emit({ type: "op", name: "ASSET", numArgs: 0, numResults: 1 })
          break
      }
      emit({ type: "push", literalType: "Number", value: "1" }) // VM version
      emit({ type: "get", variable: node.contract.program })
      emit({ type: "op", name: "CHECKOUTPUT", numArgs: 6, numResults: 1 })
      return node
    }
    case "returnStatement": {
      // no-op
      return node
    }
    case "instructionExpression": {
      node.args.slice().reverse().map(compile)
      emit({ type: "instructionOp", expression: node })
      return node
    }
    case "variable": {
      emit({ type: "get", variable: node })
      return node
    }
    case "literal": {
      emit({ type: "push", literalType: node.literalType, value: node.value })
      return node
    }
    case "listLiteral": {
      throw new BugError("list literal should have been desugared before compileToIntermediate")
    }
    case "parameter": {
      throw new BugError("parameter should not be passed to compileToIntermediate")
    }
    case "contractExpression": {
      throw new BugError("storedValue should not be passed to compileToIntermediate")
    }
    case "storedValue": {
      throw new BugError("storedValue should not be passed to compileToIntermediate")
    }
  }
}
