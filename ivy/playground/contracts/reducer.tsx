import { ContractsState } from './types'
import { SELECT_TEMPLATE } from './actions'
import { getParameterIds } from '../templates/selectors'
import { OldTemplate } from '../templates/types'
import { Input, InputMap } from '../inputs/types'
import { getInputMap } from '../templates/selectors'
import { addParameterInput } from '../inputs/data'
import { AppState } from '../app/types'
import { createSelector } from 'reselect'
import { CREATE_CONTRACT, UPDATE_CLAUSE_INPUT, UPDATE_INPUT,
         SET_CLAUSE_INDEX, SPEND_CONTRACT } from './actions'
import { addDefaultInput, getPublicKeys } from '../inputs/data'
import { Contract as Contract } from './types'
import { ClauseParameterType } from 'ivy-compiler'
import { Param } from '../templates/types'

export const INITIAL_STATE: ContractsState = {
  itemMap: {},
  idList: [],
  spentIdList: [],
  spendContractId: "",
  selectedClauseIndex: 0,
}

export function generateInputMap(contractParameters: Param[], valueParam: string): InputMap {
  let inputs: Input[] = []
  for (let parameter of contractParameters) {
    addParameterInput(inputs, parameter.type as ClauseParameterType, "contractParameters." + parameter.name)
  }
  if (valueParam !== "") {
    addParameterInput(inputs, "Value", "contractValue." + valueParam)
  }

  let inputMap = {}
  for (let input of inputs) {
    inputMap[input.name] = input
  }
  return inputMap
}

export default function reducer(state: ContractsState = INITIAL_STATE, action): ContractsState {
  switch (action.type) {
    case SPEND_CONTRACT: {
      const contract = state.itemMap[action.id]
      return {
        ...state,
        itemMap: {
          ...state.itemMap,
          [action.id]: {
            ...contract,
            lockTxid: action.lockTxid
          }
        },
        idList: state.idList.filter(id => id !== action.id),
        spentIdList: [action.id, ...state.spentIdList]
      }
    }
    case CREATE_CONTRACT: // reset keys etc. this is safe (the action already has this stuff)
      let controlProgram = action.controlProgram
      let hash = action.utxo.transactionId
      let template: OldTemplate = action.template
      let clauseNames = template.clauses.map(clause => clause.name)
      let clauseParameterIds = {}
      let inputs: Input[] = []
      for (let clause of template.clauses) {
        clauseParameterIds[clause.name] = clause.parameters.map(param => "clauseParameters." + clause.name + "." + param.identifier)
        for (let parameter of clause.parameters) {
          addParameterInput(inputs, parameter.valueType, "clauseParameters." + clause.name + "." + parameter.identifier)
        }

        for (const output of clause.outputs) {
          const inputName = "contractValue." + output.contract.value.identifier
          if (action.inputMap[inputName]) {
            // This is the locked value.
            // Do not add it to the spendInputMap.
            continue
          }
          addParameterInput(inputs, "Value", "clauseValue." + clause.name + "." + output.contract.value.identifier)
        }
      }
      addDefaultInput(inputs, "accountInput", "transactionDetails") // return destination. not always used
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
        lockTxid: '',
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
        idList: [contract.id, ...state.idList],
        itemMap: {
          ...state.itemMap,
          [contractId]: contract
        },
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
    case "@@router/LOCATION_CHANGE":
      let path = action.payload.pathname.split("/")
      if (path[1] === "ivy") {
        path.shift()
      }
      if (path.length > 2 && path[1] === "unlock") {
        return {
          ...state,
          spendContractId: path[2],
          selectedClauseIndex: 0,
        }
      }
      return state
    default:
      return state
  }
}

export const getParameterInputs = createSelector(
  getInputMap,
  getParameterIds,
  (inputMap, parameterIds) => {
    return inputMap && parameterIds && parameterIds.map(id => inputMap[id])
  }
)
