import {
  Variable
} from './ast'

import {
  Operation,
  FinalOperation,
  operationToString
} from './intermediate'

import {
  BugError
} from './errors'

type Stack = string[]

function getDepth(stack: Stack, identifier: string) {
  let reversedStack = [...stack].reverse()
  return reversedStack.indexOf(identifier)
}

function pick(stack: Stack, depth: number) {
  let item = stack[stack.length - depth - 1]
  stack.push(item)
}

function roll(stack: Stack, depth: number) {
  let item = stack[stack.length - depth - 1]
  stack.splice(stack.length - depth - 1, 1)
  stack.push(item)
}

function popMany(stack: Stack, numToPop: number) {
  for (let i = 0; i < numToPop; i++) {
    stack.pop()
  }
}

function pushResult(stack: Stack) {
  stack.push("(result)")
}

export function compileStackOps(ops: Operation[]): FinalOperation[] {

  let newOps: FinalOperation[] = []

  const emit = (op: FinalOperation) => {
    newOps.push(op)
  }

  let firstOp = ops.shift()

  if (!firstOp || firstOp.type !== "beginContract") throw new BugError("first operation must be beginContract")

  let contract = firstOp.contract

  let contractParameterNames = contract.parameters.map(param => param.identifier).reverse()

  let clauseSelector = (contract.clauseSelector && (contract.numClauses > 1)) ? [contract.clauseSelector] : []

  let defaultCounts = new Map<string, number>()

  if (contract.clauseSelector) {
    defaultCounts.set(contract.clauseSelector, contract.numClauses - 1)
  }

  let defaultStack: Stack = [...clauseSelector, ...contractParameterNames]

  let stack = defaultStack
  let counts = defaultCounts
  
  contract.parameters.slice().reverse().map(param => emit({ type: "pushParameter", identifier: param.identifier }))

  for (let op of ops) {
    switch(op.type) {
      case "beginContract": {
        throw new BugError("only one beginContract op is allowed per contract")
      }
      case "beginClause": {
        if (op.clause.referenceCounts === undefined) throw new BugError("reference counts map surprisingly undefined")
        counts = new Map(op.clause.referenceCounts)
        let clauseParameterNames = op.clause.parameters.map(param => param.identifier).reverse()
        stack = [...clauseParameterNames, ...defaultStack]
        // empty the stack of parameters that aren't used in this clause
        for (let [key, value] of op.clause.referenceCounts) {
          if (value === 0) {
            let depth = getDepth(stack, key)
            if (depth === -1) throw new BugError(key + " not found in stack")
            emit({ type: "roll", depth: depth })
            roll(stack, depth)
            emit({ type: "drop" })
            popMany(stack, 1)
          }
        }
        // remove clause selector if it's still there
        if (contract.clauseSelector && stack.indexOf(contract.clauseSelector) !== -1) {
          let depth = getDepth(stack, contract.clauseSelector)
          if (depth === -1) throw new BugError(contract.clauseSelector + " not found in stack")
          emit({ type: "roll", depth: depth })
          roll(stack, depth)
          emit({ type: "drop" })
          popMany(stack, 1)
        }
        break
      }
      case "endClause": {
        // if (stack.length !== 1) {
        //   throw new BugError("stack not empty after clause " + op.clause.name)
        // }
        counts = defaultCounts
        stack = defaultStack
        break
      }
      case "get": {
        let count = counts.get(op.variable.identifier)
        if (count === undefined) throw new BugError("reference count unexpectedly undefined")
        if (count <= 0) throw new BugError("reference count unexpectedly " + count)
        let depth = getDepth(stack, op.variable.identifier)
        if (depth === -1) throw new BugError(op.variable.identifier + " not found in stack")
        if (count === 1) {
          emit({ type: "roll", depth: depth })
          roll(stack, depth)
        } else {
          emit({ type: "pick", depth: depth })
          pick(stack, depth)
        }
        counts.set(op.variable.identifier, count - 1)
        break
      }
      case "instructionOp": {
        let numInputs = op.expression.args.length
        popMany(stack, numInputs)
        pushResult(stack)
        emit(op)
        break
      }
      case "op": {
        popMany(stack, op.numArgs)
        for (let i = 0; i < op.numResults; i++) {
          pushResult(stack)
        }
        emit(op)
        break
      }
      case "push": {
        pushResult(stack)
        emit(op)
        break
      }
      case "beginIf": {
        popMany(stack, 1)
        emit(op)
        break
      }
      default: emit(op)
    }
  }
  return newOps
}

