// external imports
import { createSelector } from 'reselect'
import { sha3_256 } from 'js-sha3'

// ivy imports
import { client, signer } from '../core'
import { AppState } from '../app/types'
import { CompiledTemplate } from '../templates/types'

import {
  Contract,
  ContractsState,
  ContractMap,
} from './types'

import {
  HashFunction,
  ClauseParameterType,
  Input,
  InputMap,
  ProgramInput,
  ValueInput
} from '../inputs/types'

import {
  isValidInput,
  getData,
  addParameterInput,
} from '../inputs/data'

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

// internal imports
import { SPEND_CONTRACT } from './actions'

export const getState = (state: AppState): ContractsState => state.contracts

export const getContractIds= createSelector(
  getState,
  (state: ContractsState) => state.idList
)

export const getSpentContractIds = createSelector(
  getState,
  (state: ContractsState) => state.spentIdList
)

export const getContractMap = createSelector(
  getState,
  (state: ContractsState) => state.contractMap
)

export const getContract = (state: AppState, contractId: string) => {
  const contractMap = getContractMap(state)
  return contractMap[contractId]
}

export const getSpendContractId = createSelector(
  getState,
  (state: ContractsState): string => state.spendContractId
)

export const getSelectedClauseIndex = createSelector(
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
  getContractMap,
  getSpendContractId,
  (contractMap: ContractMap, contractId: string) => {
    const spendContract = contractMap[contractId]
    if (spendContract === undefined)
      throw "no contract for ID " + contractId
    return spendContract
  }
)

export const getInputSelector = (id: string) => {
  return createSelector(
    getInputMap,
    (inputMap: InputMap) => {
      const input = inputMap[id]
      if (input === undefined) {
        throw "bad spend input ID: " + id
      } else {
        return input
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

export const getInputMap = createSelector(
  getSpendContract,
  spendContract => spendContract.inputMap
)

export const getParameterIds = createSelector(
  getSpendContract,
  spendContract => spendContract.template.params.map(param => "contractParameters." + param.name)
)

export const getSelectedClause = createSelector(
  getSpendContract,
  getSelectedClauseIndex,
  (spendContract, clauseIndex) => {
    return spendContract.template.clauseInfo[clauseIndex]
  }
)

export const getClauseName = createSelector(
  getSelectedClause,
  clause => clause.name
)

export const getClauseParameters = createSelector(
  getSelectedClause,
  (clause) => clause.args
)

export const getClauseParameterIds = createSelector(
  getClauseName,
  getClauseParameters,
  (clauseName, clauseParameters) => {
    return clauseParameters.map(param => "clauseParameters." + clauseName + "." + param.name)
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
  getSelectedClauseIndex,
  (spendInputMap: InputMap, clauseName: string, clauseParameters, contract, clauseIndex): WitnessComponent[] => {
    const witness: WitnessComponent[] = []
    clauseParameters.forEach(clauseParameter => {
      const clauseParameterPrefix = "clauseParameters." + clauseName + "." + clauseParameter.name
      switch (clauseParameter.type) {
        case "PublicKey": {
          const inputId = clauseParameterPrefix + ".publicKeyInput.provideStringInput"
          const input = spendInputMap[inputId]
          if (input === undefined || input.type !== "provideStringInput") {
            throw "provideStringInput surprisingly not found for PublicKey clause parameter"
          }
          witness.push({
            type: "data",
            value: dataToArgString(getData(inputId, spendInputMap))
          })
          return
        }
        case "String": {
          const inputId = clauseParameterPrefix + ".stringInput.provideStringInput"
          const input = spendInputMap[inputId]
          if (input === undefined || input.type !== "provideStringInput") {
            throw "provideStringInput surprisingly not found for String clause parameter"
          }
          witness.push({
            type: "data",
            value: dataToArgString(getData(inputId, spendInputMap))
          })
          return
        }
        case "Signature": {
          const inputId = clauseParameterPrefix + ".signatureInput.choosePublicKeyInput"
          const input = spendInputMap[inputId]
          if (input === undefined || input.type !== "choosePublicKeyInput") {
            throw "choosePublicKeyInput surprisingly not found"
          }

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
        default: {
          const val = dataToArgString(getData(clauseParameterPrefix, spendInputMap))
          witness.push({
            type: "data",
            value: val
          })
          return
        }
      }
    })
    const reverse = witness.reverse()
    if (contract.clauseList.length > 1) {
      const value = dataToArgString(clauseIndex)
      reverse.push({
        type: "data",
        value
      } as DataWitness)
    }
    return reverse
  }
)

export const getClauseValueInfo = createSelector(
  getSelectedClause,
  (clause) => {
    return clause.valueInfo
  }
)

export const getClauseUnlockInput = createSelector(
  getSelectedClause,
  getSpendInputMap,
  (clause, spendInputMap) => {
    let input
    clause.valueInfo.forEach(value => {
      if (value.program === undefined) {
        input = spendInputMap["unlockValue.accountInput"]
      }
    })
    return input
  }
)

export const getUnlockAction = createSelector(
  getSpendContract,
  getClauseUnlockInput,
  (contract, unlockInput) => {
    if (unlockInput === undefined || unlockInput.value === '') {
      return undefined
    }
    return {
        type: "controlWithAccount",
        accountId: unlockInput.value,
        assetId: contract.assetId,
        amount: contract.amount
    } as ControlWithAccount
  }
)

export const getClauseMintimes = createSelector(
  getSpendContract,
  getSelectedClauseIndex,
  (spendContract, clauseIndex) => {
    const clauseName = spendContract.clauseList[clauseIndex]
    const mintimes = spendContract.template.clauseInfo[clauseIndex].mintimes
    return mintimes.map(argName => {
      const inputMap = spendContract.inputMap
      return new Date(inputMap["contractParameters." + argName + ".timeInput.timestampTimeInput"].value)
    })
  }
)

export const getClauseMaxtimes = createSelector(
  getSpendContract,
  getSelectedClauseIndex,
  (spendContract, clauseIndex) => {
    const clauseName = spendContract.clauseList[clauseIndex]
    const maxtimes = spendContract.template.clauseInfo[clauseIndex].maxtimes
    if (maxtimes === undefined)
      return []

    return maxtimes.map(argName => {
      const inputMap = spendContract.inputMap
      return new Date(inputMap["contractParameters." + argName + ".timeInput.timestampTimeInput"].value)
    })
  }
)

export const areSpendInputsValid = createSelector(
  getSpendInputMap,
  getClauseParameterIds,
  getClauseUnlockInput,
  (spendInputMap, parameterIds, unlockInput) => {
    const invalid = parameterIds.filter(id => {
      return !isValidInput(id, spendInputMap)
    })
    return (invalid.length === 0) && (unlockInput === undefined || isValidInput('unlockValue.accountInput', spendInputMap))
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

export const getRequiredValueAction = createSelector(
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

export const getLockActions = createSelector(
  getInputMap,
  getClauseValueInfo,
  (inputMap, valueInfo) => {
    return valueInfo
      .filter(value => value.program !== undefined)
      .map(value => {
        const progName = value.program
        const progInput = inputMap["contractParameters." + progName + ".programInput"] as ProgramInput
        if (progInput === undefined) throw "programInput unexpectedly undefined"
        if (progInput.computedData === undefined) throw "programInput.computedData unexpectedly undefined"
        const receiver: Receiver = {
          controlProgram: progInput.computedData,
          expiresAt: "2020-06-25T00:00:00.000Z" // TODO
        }

        // Handles locking a contract paramater's asset amount
        let assetInput = inputMap["contractParameters." + value.asset + ".assetInput"]
        let amountInput = inputMap["contractParameters." + value.amount + ".amountInput"]

        // Handles locking a required value
        if (assetInput === undefined) {
          assetInput = inputMap["clauseValue." + value.name + ".valueInput.assetInput"]
          amountInput = inputMap["clauseValue." + value.name + ".valueInput.amountInput"]
        }

        // Handles re-locking the locked value
        if (assetInput === undefined) {
          assetInput = inputMap["contractValue." + value.name + ".valueInput.assetInput"]
          amountInput = inputMap["contractValue." + value.name + ".valueInput.amountInput"]
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

export const generateInputMap = (compiled: CompiledTemplate): InputMap => {
  let inputs: Input[] = []
  for (const param of compiled.params) {
    switch(param.type) {
      case "Sha3(PublicKey)": {
        const hashParam = {
          type: "hashType",
          inputType: "PublicKey",
          hashFunction: "sha3" as HashFunction
        }
        addParameterInput(inputs, hashParam as ClauseParameterType, "contractParameters." + param.name)
        break
      }
      case "Sha3(String)": {
        const hashParam = {
          type: "hashType",
          inputType: "String",
          hashFunction: "sha3" as HashFunction
        }
        addParameterInput(inputs, hashParam as ClauseParameterType, "contractParameters." + param.name)
        break
      }
      default:
        addParameterInput(inputs, param.type as ClauseParameterType, "contractParameters." + param.name)
    }
  }

  if (compiled.value !== "") {
    addParameterInput(inputs, "Value", "contractValue." + compiled.value)
  }

  const inputMap = {}
  for (let input of inputs) {
    inputMap[input.name] = input
  }
  return inputMap
}
