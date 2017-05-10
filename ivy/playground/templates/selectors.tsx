import { compileContractParameters } from '../../ivy-compiler/src';
import { createSelector } from 'reselect'

import * as app from '../app/types'
import { TemplateState, Template, ItemMap } from './types'
import { Input, InputMap } from '../inputs/types'
import { compileTemplate } from 'ivy-compiler'
import { mapServerTemplate } from './util'
import { SpendFromAccount } from '../transactions/types'
import { isValidInput, getData } from '../inputs/data'

export const getTemplateState = (state: app.AppState): TemplateState => state.templates

export const getSource = createSelector(
  getTemplateState,
  (state: TemplateState): string => state.source
)

export const getItemMap = createSelector(
  getTemplateState,
  (state: TemplateState): ItemMap => state.itemMap
)

export const getContractParameters = createSelector(
  getTemplateState,
  (state: TemplateState) => state.contractParameters
)

export const getIdList = createSelector(
  getTemplateState,
  state => state.idList
)

export const getItem = (id: string) => {
  return createSelector(
    getItemMap,
    itemMap => itemMap[id]
  )
}

export const getInputMap = createSelector(
  getTemplateState,
  templateState => templateState.inputMap
)

export const getInputList = createSelector(
  getInputMap,
  inputMap => {
    if (inputMap === undefined) return undefined
    let inputList: Input[] = []
    for (const id in inputMap) {
      inputList.push(inputMap[id])
    }
    return inputList
  }
)

export const getCompiled = createSelector(
  getTemplateState,
  (state) => state.compiled
)

export const getOpcodes = createSelector(
  getCompiled,
  (compiled) => compiled && compiled.opcodes
)

export const getParameterIdList = createSelector(
  getContractParameters,
  (contractParameters) => {
    return contractParameters && contractParameters
      .map(param => "contractParameters." + param.identifier)
  }
)

export const getDataParameterIdList = createSelector(
  getContractParameters,
  (contractParameters) => {
    return contractParameters && contractParameters
      .filter(param => param.valueType !== "Value" )
      .map(param => "contractParameters." + param.identifier)
  }
)

export const getParameterIds = createSelector(
  getContractParameters,
  (contractParameters) => {
    return contractParameters && contractParameters
      .map(param => "contractParameters." + param.identifier)
  }
)

export const areInputsValid = createSelector(
  getInputMap,
  getParameterIds,
  (inputMap, paramIdList) => {
    if (inputMap === undefined || paramIdList === undefined) return false
    const invalid = paramIdList.filter(id => {
      return !isValidInput(id, inputMap)
    })
    return invalid.length === 0
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

export const getParameterData = (state, inputMap) => {
  let parameterIds = getDataParameterIds(state)
  if (parameterIds === undefined) throw "parameter IDs should not be undefined when getParameterData is called"
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

export const getDataParameterIds = createSelector(
  getContractParameters,
  (contractParameters) => {
    return contractParameters && contractParameters
      .filter(param => param.valueType !== "Value" )
      .map(param => "contractParameters." + param.identifier)
  }
)

export const getSelected = createSelector(
  getCompiled,
  getItemMap,
  (compiled, itemMap) => {
    if (compiled === undefined || 
        itemMap[compiled.name] === undefined) {
      return ""
    } else {
      return compiled.name
    }
  }
)

export const getSaveability = createSelector(
  getCompiled,
  getItemMap,
  (compiled, itemMap) => {
    if (compiled === undefined) return {
      saveable: false,
      error: "Contract template has not been compiled."
    }
    if (compiled.error !== "") return {
      saveable: false,
      error: "Contract template is not valid Ivy."
    }
    let name = compiled.name
    if (itemMap[name] !== undefined) return {
      saveable: false,
      error: "There is already a contract template saved with that name."
    }
    return {
      saveable: true,
      error: ""
    }
  }
)

export const getCreateability = createSelector(
  getSource,
  getItemMap,
  getCompiled,
  getContractValue,
  areInputsValid,
  (source, itemMap, compiled, inputsAreValid, contractValue) => {
    if (compiled === undefined) return {
      createable: false,
      error: "Contract template has not been compiled."
    }
    if (compiled.error !== "") return {
      createable: false,
      error: "Contract template is not valid Ivy."
    }
    if (!inputsAreValid || contractValue === undefined) return {
      createable: false,
      error: "One or more arguments to the contract are invalid."
    }
    let name = compiled.name
    let savedSource = itemMap[name]
    if (savedSource === undefined) return {
      createable: false,
      error: "Contract template must be saved before it can be instantiated."
    }
    if (savedSource !== source) return {
      createable: false,
      error: "Contract template must be saved (under an unused name) before it can be instantiated."
    }
    return {
      createable: true,
      error: ""
    }
  }
)