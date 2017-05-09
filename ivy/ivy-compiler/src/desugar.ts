import {
  ASTNode,
  Assertion,
  Block,
  Clause,
  Conditional,
  Contract,
  ContractExpression,
  Expression,
  InstructionExpression,
  ListLiteral,
  mapOverAST,
  Parameter,
  RawContract,
  Statement
} from './ast'

import {
  BugError,
  IvyTypeError
} from './errors'

function setupClauses(oldClauses: Clause[], clauseSelector: string): Block {
  let newClauses = [...oldClauses]
  const clause = newClauses.pop()
  if (clause === undefined) throw new BugError("undefined clause")
  if (newClauses.length === 0) {
    // last clause, or only one clause
    return clause
  }
  let condition  
  if (newClauses.length === 1) {
    condition = { type: "variable", location: clause.location, identifier: clauseSelector }
  } else {
    let args: Expression[] = [{ type: "literal", 
                                literalType: "Integer",
                                location: clause.location, 
                                value: newClauses.length.toString()
                              }, {
                                type: "variable",
                                location: clause.location,
                                identifier: clauseSelector
                              }]
    condition = { type: "instructionExpression",
                  expressionType: "binaryExpression",
                  location: clause.location, 
                  instruction: "==", 
                  args: args
                }
  }

  return { 
    type: "conditional", 
    location: clause.location, 
    condition: condition, 
    ifBlock: clause, 
    elseBlock: setupClauses(newClauses, clauseSelector),
    endTag: genJumpTarget(),
    elseTag: genJumpTarget()
  }
}

function desugarAssetAmountClause(clause: Clause): Clause {
  let parameters = clause.parameters.slice()
  let assertions = clause.assertions.slice()
  let i = 0
  while (i < parameters.length) {
    let param = parameters[i]
    if (param.itemType == "AssetAmount") {
      let assetParam: Parameter = {
        ...param,
        identifier: param.identifier + ".asset"
      }
      let amountParam: Parameter = {
        ...param,
        identifier: param.identifier + ".amount"
      }
      parameters.splice(i, 1, assetParam, amountParam)
      let k = 0
      while (k < assertions.length) {
        let assertion = assertions[k]
        if (findParam(assertion.expression, param)) {
          let assetAssertion = duplicateAndReplace(assertion, param, assetParam) as Assertion
          let amountAssertion = duplicateAndReplace(assertion, param, amountParam) as Assertion
          assertions.splice(k, 1, assetAssertion, amountAssertion)
          k += 2
        } else {
          k++
        }
      }
      i += 2
    } else {
      i++
    }
  }
  return {
    ...clause,
    parameters: parameters,
    assertions: assertions
  }
}

function desugarClauses(rawContract: RawContract): Contract {
  let clauses = rawContract.clauses.map(desugarAssetAmountClause)
  let numClauses = clauses.length
  let clauseSelector = clauses.map(clause => clause.name).join("/")

  let block = setupClauses(clauses, clauseSelector)

  return {
    type: "contract",
    location: rawContract.location,
    name: rawContract.name,
    parameters: rawContract.parameters,
    block: block,
    numClauses: numClauses,
    clauseSelector: (clauseSelector === "/") ? undefined : clauseSelector
  }
}

function duplicateAndReplace(node: ASTNode, oldParam: Parameter, newParam: Parameter): ASTNode {
  return mapOverAST((node: ASTNode) => {
    if (node.type == "variable" && node.identifier == oldParam.identifier) {
      return {
        ...node,
        identifier: newParam.identifier
      }
    }
    return node
  }, node)
}

// Decorate outputs with their indexes within the clause (for use by
// CHECKOUTPUT).  For each output of a clause-passed value, ensure
// it's verified against some AssetAmount; then discard the verify and
// decorate the output with the information that will be needed (from
// that AssetAmount) for the eventual CHECKOUTPUT.
function desugarOutputs(clause: Clause, assetAmountParams: string[]): Clause {
  // This function tests whether a given expression is
  // "val.assetAmount == param" (or "param == val.assetAmount") for a
  // given Value (as a ContractExpression) and a given AssetAmount (as
  // a Parameter).
  let isAssetAmountValueEquality = function(expr: Expression, contractExpr: ContractExpression, param: string): boolean {
    if (expr.type != "instructionExpression") {
      return false
    }
    if (expr.expressionType != "binaryExpression") {
      return false
    }
    if (expr.instruction != "==") {
      return false
    }
    if (expr.args.length != 2) {
      return false
    }

    // This function tests whether an expression is "val.assetAmount"
    // (where "val" is the identifier in contractExpr)
    let isValAssetAmount = function(expr: Expression): boolean {
      return expr.type == "variable" && expr.identifier == (contractExpr.value.identifier + ".assetAmount")
    }
    if (!isValAssetAmount(expr.args[0]) && !isValAssetAmount(expr.args[1])) {
      return false
    }

    // This function tests whether an expression is "param"
    let isParam = function(expr: Expression): boolean {
      return expr.type == "variable" && expr.identifier == param
    }
    if (!isParam(expr.args[0]) && !isParam(expr.args[1])) {
      return false
    }
    return true
  }

  let assertions = clause.assertions.slice()
  for (let i = 0; i < clause.outputs.length; i++) {
    let output = clause.outputs[i]
    output.index = i
    if (output.contract.value.type != "variable") {
      // An output of type "storedValue" (i.e., the value locked by
      // the contract) does not have to be checked against an
      // assetAmount.
      continue
    }
    let j = 0
    let found = false
    while (j < assertions.length) {
      let assertion = assertions[j]
      for (let param of assetAmountParams) {
        if (isAssetAmountValueEquality(assertion.expression, output.contract, param)) {
          found = true
          // decorate the output with info that will be needed by
          // CHECKOUTPUT
          output.assetAmountParam = param
          break
        }
      }
      if (found) {
        // remove this assertion
        assertions.splice(j, 1)
        break
      }
      j++
    }
    if (!found) {
      throw new IvyTypeError("value in output in clause " + clause.name + " is not checked against an AssetAmount")
    }
  }
  return {
    ...clause,
    assertions: assertions
  }
}

export function desugarContract(rawContract: RawContract): Contract {
  let parameters = rawContract.parameters.slice(0, -1) // remove the required Value param

  // Desugar output calls and their matching assertions.  For each
  // output(val) (where val is some Value), look for an assertion in
  // the same clause matching it against some AssetAmount; then,
  // remove the assertion and replace the output call with a decorated
  // version mentioning the AssetAmount parameter.
  let assetAmountParams: string[] = []
  for (let param of parameters) {
    if (param.itemType == "AssetAmount") {
      assetAmountParams.push(param.identifier)
    }
  }
  let clauses = rawContract.clauses.map(clause => desugarOutputs(clause, assetAmountParams))

  // Desugar AssetAmount parameters.
  // In the contract parameter list, a single AssetAmount "foo"
  // becomes two params: foo.asset and foo.amount.
  // In clause bodies, any Assertion containing foo becomes two
  // assertions, one with foo.asset replacing foo, one with foo.amount
  // replacing foo.
  // (However, desugarOutputs above should have removed any assertions
  // containing AssetAmounts.)
  let i = 0
  while (i < parameters.length) {
    let param = parameters[i]
    if (param.itemType == "AssetAmount") {
      let assetParam: Parameter = {
        ...param,
        identifier: param.identifier + ".asset"
      }
      let amountParam: Parameter = {
        ...param,
        identifier: param.identifier + ".amount"
      }
      parameters.splice(i, 1, assetParam, amountParam)
      for (let j = 0; j < clauses.length; j++) {
        let clause = clauses[j]
        let assertions = clause.assertions.slice()
        let k = 0
        while (k < assertions.length) {
          let assertion = assertions[k]
          if (findParam(assertion.expression, param)) {
            let assetAssertion = duplicateAndReplace(assertion, param, assetParam) as Assertion
            let amountAssertion = duplicateAndReplace(assertion, param, amountParam) as Assertion
            assertions.splice(k, 1, assetAssertion, amountAssertion)
            k += 2
          } else {
            k++
          }
        }
        clauses[j] = {
          ...clause,
          assertions: assertions
        }
      }
      i += 2
    } else {
      i++
    }
  }
  rawContract = {
    ...rawContract,
    parameters: parameters,
    clauses: clauses
  }
  let contract = desugarClauses(rawContract)
  return mapOverAST((node: ASTNode) => {
    if (node.type == "instructionExpression") {
      switch (node.instruction) {
        case "checkTxSig": {
          // add the implicit tx sighash
          let pubkey = node.args[0]
          let sig = node.args[1]
          let sighash: Expression = {
            type: "instructionExpression",
            expressionType: "callExpression",
            instruction: "tx.sighash",
            args: [],
            location: pubkey.location
          }
          let args: Expression[] = [pubkey, sighash, sig]
          return {
              ...node,
            args: args
          }
        }
        case "checkMultiSig":
          // deconstruct the lists
          // and add the dummy 0 value
          let pubKeys = node.args[0] as ListLiteral
          let sigs = node.args[1] as ListLiteral
          let args: Expression[] = [{ type: "literal", location: pubKeys.location, literalType: "Integer", value: pubKeys.values.length.toString()}, 
                                    ...pubKeys.values,
                                    { type: "literal", location: sigs.location, literalType: "Integer", value: sigs.values.length.toString()},
                                    ...sigs.values,
                                    { type: "literal", location: node.location, literalType: "Integer", value: "0"}] // dummy 0 value
          return {
              ...node,
            args: args
          }

        default:
          return node
      }
    }
    return node
  }, contract) as Contract
}

let jumpTargetLetter = "a"
let jumpTargetNumber = 0

function genJumpTarget(): string {
  let result = jumpTargetLetter
  if (jumpTargetNumber > 0) {
    result += jumpTargetNumber.toString()
  }
  if (jumpTargetLetter == "z") {
    jumpTargetLetter = "a"
    jumpTargetNumber++
  } else {
    let codePoint = jumpTargetLetter.codePointAt(0)
    if (codePoint === undefined) throw "codePointAt unexpectedly returned undefined"
    jumpTargetLetter = String.fromCodePoint(codePoint + 1)
  }
  return result
}

function findParam(node: ASTNode, param: Parameter): boolean {
  let result = false
  mapOverAST((node: ASTNode) => {
    if (node.type == "variable" && node.identifier == param.identifier) {
      result = true
    }
    return node
  }, node)
  return result
}
