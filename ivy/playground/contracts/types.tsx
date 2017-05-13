import { Input, InputMap } from '../inputs/types'

import { OldTemplate } from '../templates/types'

export type Contract = {
  id: string,
  lockTxid: string,
  outputId: string,
  assetId: string,
  amount: number,
  template: OldTemplate,
  controlProgram: string,

  // Map of UI Form inputs
  // used during locking tx.
  inputMap: InputMap,

  // Map of UI Form inputs
  // used during unlocking tx.
  spendInputMap: InputMap,

  // Details on the contract clauses.
  clauseList: string[],
  clauseMap: {
    [s: string]: string[]
  }
}

export type ItemMap = { [s: string]: Contract }

export type ContractsState = {
  itemMap: ItemMap,
  idList: string[],
  spentIdList: string[],
  spendContractId: string,
  selectedClauseIndex: number
}

export type ContractParameterType = "PublicKey" | "String" | "Integer" | "Time" | "Boolean" |
                                    "Program" | "Asset" | "Amount" | "Value" //| ContractParameterHash

export type ContractParameter = {
  type: "contractParameter",
  valueType: ContractParameterType,
  identifier: string
}

export type ClauseParameter = {
  type: "clauseParameter",
  valueType: ClauseParameterType,
  identifier: string
}

export type ClauseParameterType = "Signature" | ContractParameterType // ClauseParameterHash | ContractParameterType
