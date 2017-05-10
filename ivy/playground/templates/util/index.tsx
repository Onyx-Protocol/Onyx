import { compileTemplate, ContractParameter, ClauseParameter, TemplateClause } from 'ivy-compiler'

import { Template, CompilerResult } from '../types'

export const mapServerTemplate = (tpl): Template => {
  const clauses: TemplateClause[] = tpl.clauseInfo.map(clause => {
    const parameters: ClauseParameter[] = clause.args.map(param => ({
      type: "clauseParameter",
      valueType: param.type,
      identifier: param.name
    }))

    let returnStatement
    let outputs = clause.valueInfo.filter(value => {
      if (value.program === undefined) {
        // return statement
        // TODO(boymanjor): detect and handle variable return statements
        returnStatement = {
          type: "returnStatement",
          value: {
            type: "storedValue",
            identifier: value.name
          }
        }
        return false
      }
      return true
    })

    outputs = outputs.map((output, idx) => {
      return {
        type: "output",
        contract: {
          type: "contractExpression",
          address: {
            type: "variable",
            identifier: output.program
          },
          value: {
            type: "storedValue",
            identifier: output.name
          },
        },
        assetAmountParam: output.assetAmount,
        index: idx
      }
    })

    return {
      type: "templateClause",
      name: clause.name,
      parameters,
      outputs,
      mintimes: clause.mintimes,
      maxtimes: clause.maxtimes,
      returnStatement
    }
  })

  const contractParameters: ContractParameter[] = tpl.params.map(param => ({
    type: "contractParameter",
    valueType: param.type,
    identifier: param.name
  }) as ContractParameter)

  return {
    name: tpl.name,
    instructions: tpl.opcodes.split(" "),
    source: tpl.source,
    contractParameters,
    clauses
  } as Template
}
