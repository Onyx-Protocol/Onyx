import { Input, InputMap } from '../inputs/types'

import { OldTemplate } from '../templates/types'

export type Contract = {
  // lock txid
  id: string,

  // unlock txid
  lockTxid: string,

  // output id of the contract
  outputId: string,

  assetId: string,

  amount: number,

  // contract template
  template: OldTemplate,

  controlProgram: string,

  // map of parameter inputs by id
  inputMap: InputMap,

  // map of spend contract parameters
  spendInputMap: InputMap,

  // list of clauses for the contract template
  clauseList: string[],

  // map of clause names to parameter ids
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
