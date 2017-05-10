import {
  ContractParameter, ClauseParameter, toClauseParameter
} from './cvm/parameters'

import {
  Clause, Output, Return
} from './ast'

export type Template = {
  type: "template",
  source: string,
  name: string,
  clauses: TemplateClause[],
  instructions: string[],
  contractParameters: ContractParameter[]
}

export type CompilerError = {
  type: "compilerError",
  source: string,
  message: string
}

export type TemplateClause = {
  type: "templateClause",
  name: string,
  parameters: ClauseParameter[],
  outputs: Output[],
  mintimes?: string[],
  maxtimes?: string[],
  returnStatement?: Return
}

export function toTemplateClause(clause: Clause): TemplateClause {
  let clauseParameters = clause.parameters.map(toClauseParameter)
  return {
    type: "templateClause",
    name: clause.name,
    parameters: clauseParameters,
    outputs: clause.outputs,
    returnStatement: clause.returnStatement
  }
}

export function countTemplateParams(template: Template): number {
  let result = 0
  for (let p of template.contractParameters) {
    if (p.valueType != "Value") {
      result++
    }
    if (p.valueType === "AssetAmount") {
      result++ // extra increment because this is desugared to two arguments
    }
  }
  return result
}
