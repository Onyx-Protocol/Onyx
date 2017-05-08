import { ContractsState } from './types'
import { SELECT_TEMPLATE } from './actions'
import { getParameterIdList } from '../templates/selectors'
import { Item as Template } from '../templates/types'
import { Input, InputMap } from '../inputs/types'
import { getInputMap } from './selectors'
import { addParameterInput } from '../inputs/data'
import { AppState } from '../app/types'
import { createSelector } from 'reselect'
import { CREATE_CONTRACT, UPDATE_CLAUSE_INPUT, UPDATE_INPUT, 
         SET_CLAUSE_INDEX, SHOW_ERRORS } from './actions'
import { addDefaultInput, getPublicKeys } from '../inputs/data'
import { Item as Contract } from './types'

export const INITIAL_STATE: ContractsState = {
  itemMap: {},
  idList: [],
  inputMap: {},
  selectedTemplateId: "",
  spendContractId: "",
  selectedClauseIndex: 0,
  showErrors: false
}

export default function reducer(state: ContractsState = INITIAL_STATE, action): ContractsState {
  switch (action.type) {
    case UPDATE_INPUT:
      let name = action.name
      let newValue = action.newValue
      return {
        ...state,
        inputMap: {
          ...state.inputMap,
          [name]: {
            ...state.inputMap[name],
            value: newValue
          }
        }
      }
    case SELECT_TEMPLATE:
      return {
        ...state,
        inputMap: generateInputMap(action.template),
        selectedTemplateId: action.templateId,
        showErrors: false
      }
    case CREATE_CONTRACT: // reset keys etc. this is safe (the action already has this stuff)
      let controlProgram = action.controlProgram
      let hash = action.utxo.transactionId
      let template: Template = action.template
      let clauseNames = template.clauses.map(clause => clause.name)
      let clauseParameterIds = {}
      let inputs: Input[] = []
      for (let clause of template.clauses) {
        clauseParameterIds[clause.name] = clause.parameters.map(param => "clauseParameters." + clause.name + "." + param.identifier)
        for (let parameter of clause.parameters) {
          addParameterInput(inputs, parameter.valueType, "clauseParameters." + clause.name + "." + parameter.identifier)
        }
      }
      // addDefaultInput(inputs, "addressInput", "transactionDetails")
      // addDefaultInput(inputs, "mintimeInput", "transactionDetails")
      // addDefaultInput(inputs, "maxtimeInput", "transactionDetails")
      addDefaultInput(inputs, "accountAliasInput", "transactionDetails") // return destination. not always used
      let spendInputMap = {}
      let keyMap = getPublicKeys(action.inputMap)
      for (let input of inputs) {
        spendInputMap[input.name] = input
        if (input.type === "choosePublicKeyInput") {
          input.keyMap = keyMap
        }
      }
      let contract: Contract = {
        template: action.template,
        id: hash,
        outputId: action.utxo.id,
        assetId: action.utxo.assetId,
        amount: action.utxo.amount,
        inputMap: action.inputMap,
        controlProgram: controlProgram,
        clauseList: clauseNames,
        clauseMap: clauseParameterIds,
        spendInputMap: spendInputMap
      }
      let contractId = contract.id
      return {
        ...state,
        idList: [...state.idList, contract.id],
        itemMap: {
          ...state.itemMap,
          [contractId]: contract
        },
        inputMap: generateInputMap(action.template),
        showErrors: false
      }
    case UPDATE_CLAUSE_INPUT: {
      // gotta find a way to make this logic shorter
      // maybe further normalizing it; maybe Immutable.js or cursors or something
      let contractId = action.contractId as string
      let oldContract = state.itemMap[action.contractId]
      let oldSpendInputMap = oldContract.spendInputMap
      let oldInput = oldSpendInputMap[action.name]
      if (oldInput === undefined) throw "unexpectedly undefined clause input"
      let newInput = {
        ...oldInput,
        value: action.newValue
      }
      let newSpendInputMap = {
        ...oldSpendInputMap,
        [action.name]: newInput
      }
      newSpendInputMap[action.name] = newInput
      return {
        ...state,
        itemMap: {
          ...state.itemMap,
          [action.contractId]: {
            ...oldContract,
            spendInputMap: newSpendInputMap
          }
        }
      }
    }
    case SET_CLAUSE_INDEX: {
      return {
        ...state,
        selectedClauseIndex: action.selectedClauseIndex
      }
    }
    case SHOW_ERRORS: {
      return {
        ...state,
        showErrors: true
      }
    }
    case "@@router/LOCATION_CHANGE":
      let path = action.payload.pathname.split("/")
      if (path[1] === "spend") {
        return {
          ...state,
          spendContractId: path[2],
          selectedClauseIndex: 0,
          showErrors: false
        }
      }
      return state
    default:
      return state
  }
}

function generateInputMap(template: Template): InputMap {
  if (template === undefined) {
    return {}
  }

  let inputs: Input[] = []
  for (let parameter of template.contractParameters) {
    addParameterInput(inputs, parameter.valueType, "contractParameters." + parameter.identifier)
  }

  let inputMap = {}
  for (let input of inputs) {
    inputMap[input.name] = input
  }
  return inputMap
}

export const getParameterInputs = createSelector(
  getInputMap,
  getParameterIdList,
  (inputMap, parameterIdList) => {
    return parameterIdList.map(id => inputMap[id])
  }
)
