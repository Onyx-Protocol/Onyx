import { SPEND_CONTRACT } from './actions';
import { createSelector } from 'reselect'
import { Template } from '../templates/types'

import * as app from '../app/types'

import { client, signer } from '../util'

import {
  Item as Contract,
  ItemMap,
  ContractsState,
} from './types'

import {
  Input,
  InputMap,
  AddressInput,
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
} from '../transactions/types';

import {
  getItemMap as getTemplateMap
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
        case "AssetAmount": // TODO
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
    const returnInput = spendInputMap["transactionDetails.accountAliasInput"]
    return {
        type: "controlWithAccount",
        accountId: returnInput.value,
        assetId: contract.assetId,
        amount: contract.amount
    } as ControlWithAccount
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

export const getClauseValues = createSelector(
  getSpendTemplateClause,
  getSpendInputMap,
  (clause, spendInputMap) => {
    return clause.parameters
      .filter(param => param.valueType === "Value")
      .map(param => {
        let clauseParameterPrefix = "clauseParameters." + clause.name + "." + param.identifier
        let accountInput = spendInputMap[clauseParameterPrefix + ".valueInput.accountAliasInput"]
        let assetInput = spendInputMap[clauseParameterPrefix + ".valueInput.assetAmountInput.assetAliasInput"]
        let amountInput = spendInputMap[clauseParameterPrefix + ".valueInput.assetAmountInput.amountInput"]
        if (accountInput === undefined) throw "accountInput for clause Value parameter surprisingly undefined"
        if (assetInput === undefined) throw "assetInput for clause Value parameter surprisingly undefined"
        if (assetInput === undefined) throw "assetInput for clause Value parameter surprisingly undefined"
        let amount = parseInt(amountInput.value, 10)
        let spendFromAccount: SpendFromAccount = {
          type: "spendFromAccount",
          accountId: accountInput.value,
          assetId: assetInput.value,
          amount: amount
        }
        return spendFromAccount
    })
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
    return (invalid.length === 0) && (spendTemplateClause.returnStatement === undefined || isValidInput('transactionDetails.accountAliasInput', spendInputMap))
  }
)

export const getClauseOutputActions = createSelector(
  getSpendContract,
  getClauseOutputs,
  (contract, clauseOutputs) => {
    let inputMap = contract.inputMap
    return clauseOutputs.map(clauseOutput => {
      const addressIdentifier = clauseOutput.contract.address.identifier
      const addressInput = inputMap["contractParameters." + addressIdentifier + ".addressInput"] as AddressInput
      if (addressInput === undefined) throw "addressInput unexpectedly undefined"
      if (addressInput.computedData === undefined) throw "addressInput.computedData unexpectedly undefined"
      const receiver: Receiver = {
        controlProgram: addressInput.computedData,
        expiresAt: "2020-06-25T00:00:00.000Z" // TODO
      }

      let amountInput
      let assetAliasInput
      if (clauseOutput.assetAmountParam === undefined) {
        const valueIdentifier = clauseOutput.contract.value.identifier
        assetAliasInput = inputMap["contractParameters." + valueIdentifier + ".valueInput.assetAmountInput.assetAliasInput"]
        amountInput = inputMap["contractParameters." + valueIdentifier + ".valueInput.assetAmountInput.amountInput"]
      } else {
        let assetAmountParam = clauseOutput.assetAmountParam
        amountInput = inputMap["contractParameters." + assetAmountParam + ".assetAmountInput.amountInput"]
        assetAliasInput = inputMap["contractParameters." + assetAmountParam + ".assetAmountInput.assetAliasInput"]
      }

      let action: ControlWithReceiver = {
        type: "controlWithReceiver",
        assetId: assetAliasInput.value,
        amount: parseInt(amountInput.value, 10),
        receiver
      }
      return action
    })
  }
)
