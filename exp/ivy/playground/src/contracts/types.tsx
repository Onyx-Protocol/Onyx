import { Input, InputMap } from '../inputs/types'

import { CompiledTemplate } from '../templates/types'

export type Contract = {
  // lock tx id
  id: string,
  unlockTxid: string,
  outputId: string,
  assetId: string,
  amount: number,
  template: CompiledTemplate,
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

export type ContractMap = { [s: string]: Contract }

export type ContractsState = {
  contractMap: ContractMap,
  firstTime: boolean,
  idList: string[],
  spentIdList: string[],
  spendContractId: string,
  selectedClauseIndex: number,
  isCalling: boolean,
  showUnlockInputErrors: boolean,
  error?
}

