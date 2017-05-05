import { Input, InputMap } from '../inputs/types'

import { Item as Template } from '../templates/types'

export type Item = {
  // funding tx id
  id: string,

  // output id of the contract
  outputId: string,

  assetId: string,

  amount: number,

  // contract template
  template: Template,

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

export type ItemMap = { [s: string]: Item }

export type ContractsState = {
  itemMap: ItemMap,
  idList: string[],
  inputMap: InputMap,
  selectedTemplateId: string,
  spendContractId: string,
  selectedClauseIndex: number,
  showErrors: boolean
}
