import { createSelector } from 'reselect'
import { Item as Template } from '../templates/types'

import * as app from '../app/types'

import {
  Item as Contract,
  ItemMap,
  ContractsState,
} from './types'

import {
  Input,
  InputMap
} from '../inputs/types'

import {
  isValidInput,
  getData
} from '../inputs/data'

import {
  instantiate
} from 'ivy-compiler'

import {
  KeyId,
  SignatureWitness,
  DataWitness,
  SpendFromAccount,
  WitnessComponent
} from '../transactions/types'

import {
  getItemMap as getTemplateMap
} from '../templates/selectors'

import templates from '../templates'

export const getState = (state: app.AppState): ContractsState => state.contracts

export const getSelectedTemplateId = createSelector(
  getState,
  (state: ContractsState) => {
    return state.selectedTemplateId
  }
)

export const getSelectedTemplate = createSelector(
  getSelectedTemplateId,
  getTemplateMap,
  (templateId: string, templateMap) => {
    return templateMap[templateId]
  }
)

export const getIdList = createSelector(
  getState,
  (state: ContractsState) => state.idList
)

export const getItemMap = createSelector(
  getState,
  (state: ContractsState) => state.itemMap
)

export const getItem = (state: app.AppState, contractId: string) => {
  let itemMap = getItemMap(state)
  return itemMap[contractId]
}

export const getInputMap = createSelector(
  getState,
  (state: ContractsState): InputMap => state.inputMap
)

export const getInputList = createSelector(
  getInputMap,
  (inputMap: InputMap): Input[] => {
    let inputList: Input[] = []
    for (const id in inputMap) {
      inputList.push(inputMap[id])
    }
    return inputList
  }
)

export const getSpendContractId = createSelector(
  getState,
  (state: ContractsState): string => state.spendContractId
)

export const getSpendContractSelectedClauseIndex = createSelector(
  getState,
  (state: ContractsState): number => state.selectedClauseIndex
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

export const getClauseParameterIds = createSelector(
  getSpendContract,
  getSpendContractSelectedClauseIndex,
  (spendContract, clauseIndex) => {
    let clauseName = spendContract.clauseList[clauseIndex]
    return spendContract.template.clauses[clauseIndex].parameters.map(param => "clauseParameters." + clauseName + "." + param.identifier)
  }
)

export const getClauseWitnessComponents = createSelector(
  getSpendInputMap,
  getClauseParameterIds,
  (spendInputMap: InputMap, clauseIds: string[]): WitnessComponent[] => {
    const witness: WitnessComponent[] = []
    clauseIds.forEach(clauseId => {
      for (const inputId in spendInputMap) {
        const input = spendInputMap[inputId]
        if (input.name.includes(clauseId)) {
          switch(input.type) {
            case "choosePublicKeyInput":
              const pubkey = input.value
              if (input.keyMap === undefined) {
                throw 'undefined keymap for input type ' + input.type
              }
              const keymap = input.keyMap[pubkey]
              witness.push({
                type: "signature",
                quorum: 1,
                keys: [{
                  xpub: keymap.rootXpub,
                  derivationPath: keymap.pubkeyDerivationPath
                } as KeyId],
                signatures: []
              } as SignatureWitness)
              break
            default:
              break
          }
        }
      }
    })
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

export const getContractValue = createSelector(
  getInputMap,
  getInputList,
  (inputMap: InputMap, inputList: Input[]): SpendFromAccount|undefined => {
    let sources: SpendFromAccount[] = []
    inputList.forEach(input => {
      if (input.type === "valueInput") {
        let inputName = input.name
        let accountId = inputMap[inputName + ".accountAliasInput"].value
        let assetId = inputMap[inputName + ".assetAmountInput.assetAliasInput"].value
        let amount = parseInt(inputMap[inputName + ".assetAmountInput.amountInput"].value, 10)
        if (isNaN(amount) || amount < 0 || !accountId || !assetId) {
          return []
        }
        sources.push({
          type: "spendFromAccount",
          accountId: accountId,
          assetId: assetId,
          amount: amount
        } as SpendFromAccount)
      }
    })
    if (sources.length !== 1) return undefined
    return sources[0]
  }
)

export const getParameterIds = createSelector(
  getSelectedTemplate,
  (template: Template) => {
    return template.contractParameters
      .map(param => "contractParameters." + param.identifier)
  }
)

export const isValid = createSelector(
  getInputMap,
  getParameterIds,
  (inputMap, paramIdList) => {
    const invalid = paramIdList.filter(id => {
      !isValidInput(id, inputMap)
    })
    return invalid.length === 0
  }
)

export const getDataParameterIds = createSelector(
  getSelectedTemplate,
  (template: Template) => {
    return template.contractParameters
      .filter(param => param.valueType !== "Value" )
      .map(param => "contractParameters." + param.identifier)
  }
)

export const getInstructions = createSelector(
  getSelectedTemplate,
  (template) => template.instructions
)

export const getSelectedSource = createSelector(
  getSelectedTemplate,
  template => template.source
)

export const getParameterData = (state, inputMap) => {
  let parameterIds = getDataParameterIds(state)
  try {
    let parameterData: (number|Buffer)[] = []
    for (let id of parameterIds) {
      if (inputMap[id].value === "assetAmountInput") {
        let name = inputMap[id].name
        parameterData.push(getData(name + ".assetAmountInput.assetAliasInput", inputMap))
        parameterData.push(getData(name + ".assetAmountInput.amountInput", inputMap))        
      } else {
        parameterData.push(getData(id, inputMap))
      }
    }
    return parameterData.reverse()
  } catch (e) {
    console.log(e)
    return []
  }
}

export function getControlProgram(state, inputsById) {
  let template = getSelectedTemplate(state)
  let parameterData = getParameterData(state, inputsById)
  let rawScript = instantiate(template, parameterData)
  return rawScript.toString("hex")
}

export const getShowErrors = createSelector(
  getState,
  (contractsState: ContractsState) => contractsState.showErrors
)
