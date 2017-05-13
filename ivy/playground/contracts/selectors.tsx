import { SPEND_CONTRACT } from './actions';
import { createSelector } from 'reselect'
import { OldTemplate } from '../templates/types'

import * as app from '../app/types'

import { client, signer } from '../core'

import {
  Contract,
  ItemMap,
  ContractsState,
} from './types'

import {
  Input,
  InputMap,
  ProgramInput,
  ValueInput
} from '../inputs/types'

import {
  isValidInput,
  getData
} from '../inputs/data'

import {
  instantiate
} from 'ivy-compiler'

import {
  ControlWithReceiver,
  ControlWithAccount,
  DataWitness,
  KeyId,
  Receiver,
  RawTxSignatureWitness,
  SpendFromAccount,
  WitnessComponent
} from '../core/types';

import {
  getSourceMap
} from '../templates/selectors'

import templates from '../templates'

export const getState = (state: app.AppState): ContractsState => state.contracts

export const getIdList = createSelector(
  getState,
  (state: ContractsState) => state.idList
)

export const getSpentIdList = createSelector(
  getState,
  (state: ContractsState) => state.spentIdList
)

export const getItemMap = createSelector(
  getState,
  (state: ContractsState) => state.itemMap
)

export const getItem = (state: app.AppState, contractId: string) => {
  let itemMap = getItemMap(state)
  return itemMap[contractId]
}

export const getSpendContractId = createSelector(
  getState,
  (state: ContractsState): string => state.spendContractId
)

export const getSpendContractSelectedClauseIndex = createSelector(
  getState,
  (state: ContractsState): number => {
    let selectedClauseIndex = state.selectedClauseIndex
    if (typeof selectedClauseIndex === "number") {
      return selectedClauseIndex
    } else {
      return parseInt(selectedClauseIndex, 10)
    }
  }
)

export const getSpendContract = createSelector(
  getItemMap,
  getSpendContractId,
  (itemMap: ItemMap, contractId: string) => {
    let spendContract = itemMap[contractId]
    if (spendContract === undefined)
      throw "no contract for ID " + contractId
    return spendContract
  }
)

export const getSpendContractParameterSelector = (id: string) => {
  return createSelector(
    getSpendContract,
    (spendContract: Contract) => {
      let spendInput = spendContract.inputMap[id]
      if (spendInput === undefined) {
        throw "bad spend input ID: " + id
      } else {
        return spendInput
      }
    }
  )
}

export const getSpendInputSelector = (id: string) => {
  return createSelector(
    getSpendInputMap,
    (spendInputMap: InputMap) => {
     let spendInput = spendInputMap[id]
     if (spendInput === undefined) {
       throw "bad spend input ID: " + id
     } else {
       return spendInput
     }
    }
  )
}

export const getSpendInputMap = createSelector(
  getSpendContract,
  spendContract => spendContract.spendInputMap
)

export const getSpendContractParametersInputMap = createSelector(
  getSpendContract,
  spendContract => spendContract.inputMap
)

export const getSpendParameterIds = createSelector(
  getSpendContract,
  spendContract => spendContract.template.contractParameters.map(param => "contractParameters." + param.identifier)
)

export const getSpendTemplateClause = createSelector(
  getSpendContract,
  getSpendContractSelectedClauseIndex,
  (spendContract, clauseIndex) => {
    return spendContract.template.clauses[clauseIndex]
  }
)

export const getClauseParameters = createSelector(
  getSpendTemplateClause,
  (clause) => clause.parameters
)

export const getClauseName = createSelector(
  getSpendTemplateClause,
  clause => clause.name
)

export const getClauseParameterIds = createSelector(
  getClauseName,
  getClauseParameters,
  (clauseName, clauseParameters) => {
    return clauseParameters.map(param => "clauseParameters." + clauseName + "." + param.identifier)
  }
)

export function dataToArgString(data: number | Buffer): string {
  if (typeof data === "number") {
    let buf = Buffer.alloc(8)
    buf.writeIntLE(data, 0, 8)
    return buf.toString("hex")
  } else {
    return data.toString("hex")
  }
}

export const getClauseWitnessComponents = createSelector(
  getSpendInputMap,
  getClauseName,
  getClauseParameters,
  getSpendContract,
  getSpendContractSelectedClauseIndex,
  (spendInputMap: InputMap, clauseName: string, clauseParameters, contract, clauseIndex): WitnessComponent[] => {
    const witness: WitnessComponent[] = []
    clauseParameters.forEach(clauseParameter => {
      let clauseParameterPrefix = "clauseParameters." + clauseName + "." + clauseParameter.identifier
      switch (clauseParameter.valueType) {
        case "Value": {
          return
        }
        case "Signature": {
          let inputId = clauseParameterPrefix + ".signatureInput.choosePublicKeyInput"
          let input = spendInputMap[inputId]
          if (input === undefined || input.type !== "choosePublicKeyInput") throw "choosePublicKeyInput surprisingly not found"
          const pubkey = input.value
          if (input.keyMap === undefined) {
            throw 'surprisingly undefined keymap for input ' + input.name
          }
          const keymap = input.keyMap[pubkey]
          witness.push({
            type: "raw_tx_signature",
            quorum: 1,
            keys: [{
              xpub: keymap.rootXpub,
              derivationPath: keymap.pubkeyDerivationPath
            } as KeyId],
            signatures: []
          } as RawTxSignatureWitness)
          signer.addKey(keymap.rootXpub, client.mockHsm.signerConnection)
          return
        }
        case "Asset": // TODO
        case "Amount": // TODO
        default: {
          let val = dataToArgString(getData(clauseParameterPrefix, spendInputMap))
          witness.push({
            type: "data",
            value: val
          })
          return // TODO: handle
        }
      }
    })
    if (contract.clauseList.length > 1) {
      let value = dataToArgString(clauseIndex)
      witness.push({
        type: "data",
        value
      } as DataWitness)
    }
    return witness
  }
)

export const getClauseOutputs = createSelector(
  getSpendContract,
  getSpendContractSelectedClauseIndex,
  (spendContract, clauseIndex) => {
    return spendContract.template.clauses[clauseIndex].outputs
  }
)

export const getClauseReturnStatement = createSelector(
  getSpendContract,
  getSpendContractSelectedClauseIndex,
  (spendContract, clauseIndex) => {
    return spendContract.template.clauses[clauseIndex].returnStatement
  }
)

export const getClauseReturnAction = createSelector(
  getSpendContract,
  getSpendInputMap,
  getClauseReturnStatement,
  (contract, spendInputMap, returnStatement) => {
    if (returnStatement === undefined) {
      return undefined
    }
    const returnInput = spendInputMap["transactionDetails.accountInput"]
    return {
        type: "controlWithAccount",
        accountId: returnInput.value,
        assetId: contract.assetId,
        amount: contract.amount
    } as ControlWithAccount
  }
)

export const getClauseMintimes = createSelector(
  getSpendContract,
  getSpendContractSelectedClauseIndex,
  (spendContract, clauseIndex) => {
    const clauseName = spendContract.clauseList[clauseIndex]
    const mintimes = spendContract.template.clauses[clauseIndex].mintimes
    if (mintimes === undefined)
      return []

    return mintimes.map(argName => {
      const inputMap = spendContract.inputMap
      return new Date(inputMap["contractParameters." + argName + ".timeInput.timestampTimeInput"].value)
    })
  }
)

export const getClauseMaxtimes = createSelector(
  getSpendContract,
  getSpendContractSelectedClauseIndex,
  (spendContract, clauseIndex) => {
    const clauseName = spendContract.clauseList[clauseIndex]
    const maxtimes = spendContract.template.clauses[clauseIndex].maxtimes
    if (maxtimes === undefined)
      return []

    return maxtimes.map(argName => {
      const inputMap = spendContract.inputMap
      return new Date(inputMap["contractParameters." + argName + ".timeInput.timestampTimeInput"].value)
    })
  }
)

export const getClauseDataParameterIds = createSelector(
  getSpendContract,
  getSpendContractSelectedClauseIndex,
  (spendContract, clauseIndex) => {
    let clauseName = spendContract.clauseList[clauseIndex]
    return spendContract.template.clauses[clauseIndex].parameters
      .filter(param => param.valueType !== "Value")
      .map(param => "clauseParameters." + clauseName + "." + param.identifier)
  }
)

export const areSpendInputsValid = createSelector(
  getSpendInputMap,
  getClauseParameterIds,
  getSpendTemplateClause,
  (spendInputMap, paramIdList, spendTemplateClause) => {
    const invalid = paramIdList.filter(id => {
      return !isValidInput(id, spendInputMap)
    })
    return (invalid.length === 0) && (spendTemplateClause.returnStatement === undefined || isValidInput('transactionDetails.accountInput', spendInputMap))
  }
)

export const getClauseValueId = createSelector(
  getSpendInputMap,
  getClauseName,
  (spendInputMap, clauseName) => {
    for (const id in spendInputMap) {
      const input = spendInputMap[id]
      const inputClauseName = input.name.split('.')[1]
      if (clauseName === inputClauseName && input.value === "valueInput") {
        return input.name
      }
    }
    return undefined
  }
)

export const getClauseValue = createSelector(
  getClauseValueId,
  getSpendInputMap,
  (clauseValuePrefix, spendInputMap) => {
    const accountInput = spendInputMap[clauseValuePrefix + ".valueInput.accountInput"]
    if (accountInput === undefined) {
      return undefined
    }

    const assetInput = spendInputMap[clauseValuePrefix + ".valueInput.assetInput"]
    const amountInput = spendInputMap[clauseValuePrefix + ".valueInput.amountInput"]
    const amount = parseInt(amountInput.value, 10)
    const spendFromAccount: SpendFromAccount = {
      type: "spendFromAccount",
      accountId: accountInput.value,
      assetId: assetInput.value,
      amount: amount
    }
    return spendFromAccount
  }
)

export const getClauseOutputActions = createSelector(
  getSpendContract,
  getClauseOutputs,
  (contract, clauseOutputs) => {
    let inputMap = contract.inputMap
    return clauseOutputs.map(clauseOutput => {
      const programIdentifier = clauseOutput.contract.program.identifier
      const programInput = inputMap["contractParameters." + programIdentifier + ".programInput"] as ProgramInput
      if (programInput === undefined) throw "programInput unexpectedly undefined"
      if (programInput.computedData === undefined) throw "programInput.computedData unexpectedly undefined"
      const receiver: Receiver = {
        controlProgram: programInput.computedData,
        expiresAt: "2020-06-25T00:00:00.000Z" // TODO
      }

      const valueIdentifier = clauseOutput.contract.value.identifier
      let assetInput = inputMap["contractParameters." + clauseOutput.asset + ".assetInput"]
      let amountInput = inputMap["contractParameters." + clauseOutput.amount + ".amountInput"]
      if (assetInput === undefined) {
        assetInput = inputMap["clauseValue." + valueIdentifier + ".valueInput.assetInput"]
        amountInput = inputMap["clauseValue." + valueIdentifier + ".valueInput.amountInput"]
      }
      if (assetInput === undefined) {
        assetInput = inputMap["contractValue." + valueIdentifier + ".valueInput.assetInput"]
        amountInput = inputMap["contractValue." + valueIdentifier + ".valueInput.amountInput"]
      }
      const action: ControlWithReceiver = {
        type: "controlWithReceiver",
        assetId: assetInput.value,
        amount: parseInt(amountInput.value, 10),
        receiver
      }
      return action
    })
  }
)
