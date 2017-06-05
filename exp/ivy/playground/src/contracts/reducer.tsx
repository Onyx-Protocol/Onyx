// external imports
import { ContractsState } from './types'
import { getParameterIds } from '../templates/selectors'
import { CompiledTemplate } from '../templates/types'
import { ClauseParameterType, Input, InputMap, Hash } from '../inputs/types'
import { getInputMap } from '../templates/selectors'
import { addParameterInput } from '../inputs/data'

// ivy imports
import { AppState } from '../app/types'
import { addDefaultInput, getPublicKeys } from '../inputs/data'
import { Contract } from './types'

// internal imports
import { CREATE_CONTRACT, SPEND_CONTRACT, UPDATE_UNLOCK_ERROR, UPDATE_IS_CALLING,
         SHOW_UNLOCK_INPUT_ERRORS, UPDATE_CLAUSE_INPUT, SET_CLAUSE_INDEX, CLOSE_MODAL } from './actions'

export const INITIAL_STATE: ContractsState = {
  contractMap: {},
  firstTime: true,
  idList: [],
  spentIdList: [],
  spendContractId: "",
  selectedClauseIndex: 0,
  isCalling: false,
  showUnlockInputErrors: false,
  error: undefined
}

export default function reducer(state: ContractsState = INITIAL_STATE, action): ContractsState {
  switch (action.type) {
    case SPEND_CONTRACT: {
      const contract = state.contractMap[action.id]
      return {
        ...state,
        contractMap: {
          ...state.contractMap,
          [action.id]: {
            ...contract,
            unlockTxid: action.unlockTxid
          }
        },
        idList: state.idList.filter(id => id !== action.id),
        spentIdList: [action.id, ...state.spentIdList],
        error: undefined
      }
    }
    case CREATE_CONTRACT: // reset keys etc. this is safe (the action already has this stuff)
      const controlProgram = action.controlProgram
      const hash = action.utxo.transactionId
      const template: CompiledTemplate = {
        ...action.template,
        source: action.source
      }
      const clauseNames = template.clauses.map(clause => clause.name)
      const clauseParameterIds = {}
      const inputs: Input[] = []
      for (const clause of template.clauses) {
        clauseParameterIds[clause.name] = clause.params.map(param => "clauseParameters." + clause.name + "." + param.name)
        for (let param of clause.params) {
          switch(param.declaredType) {
            case "Hash": {
              addParameterInput(inputs, { type: param.declaredType, hashType: param.declaredType } as ClauseParameterType, "clauseParameters." + clause.name + "." + param.name)
            }
            default:
              addParameterInput(inputs, param.declaredType as ClauseParameterType, "clauseParameters." + clause.name + "." + param.name)
          }
        }

        for (const value of clause.values) {
          if (value.name === template.value) {
            // This is the unlock statement.
            // Do not add it to the spendInputMap.
            continue
          }
          addParameterInput(inputs, "Value", "clauseValue." + clause.name + "." + value.name)
        }
      }
      addDefaultInput(inputs, "accountInput", "unlockValue") // Unlocked value destination. Not always used.
      const spendInputMap = {}
      const keyMap = getPublicKeys(action.inputMap)
      for (const input of inputs) {
        spendInputMap[input.name] = input
        if (input.type === "choosePublicKeyInput") {
          input.keyMap = keyMap
        }
      }
      const contract: Contract = {
        template,
        id: hash,
        unlockTxid: '',
        outputId: action.utxo.id,
        assetId: action.utxo.assetId,
        amount: action.utxo.amount,
        inputMap: action.inputMap,
        controlProgram: controlProgram,
        clauseList: clauseNames,
        clauseMap: clauseParameterIds,
        spendInputMap: spendInputMap
      }
      return {
        ...state,
        idList: [contract.id, ...state.idList],
        contractMap: {
          ...state.contractMap,
          [contract.id]: contract
        },
        error: undefined
      }
    case UPDATE_CLAUSE_INPUT: {
      // gotta find a way to make this logic shorter
      // maybe further normalizing it; maybe Immutable.js or cursors or something
      let contractId = action.contractId as string
      let oldContract = state.contractMap[action.contractId]
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
        contractMap: {
          ...state.contractMap,
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
        selectedClauseIndex: action.selectedClauseIndex,
        error: undefined,
        showUnlockInputErrors: false
      }
    }
    case UPDATE_UNLOCK_ERROR: {
      return {
        ...state,
        error: action.error
      }
    }
    case CLOSE_MODAL: {
      return {
        ...state,
        firstTime: false
      }
    }
    case SHOW_UNLOCK_INPUT_ERRORS: {
      return {
        ...state,
        showUnlockInputErrors: action.result
      }
    }
    case UPDATE_IS_CALLING: {
      return {
        ...state,
        isCalling: action.isCalling
      }
    }
    case "@@router/LOCATION_CHANGE":
      const path = action.payload.pathname.split("/")
      if (path[1] === "ivy") {
        path.shift()
      }
      if (path.length > 2 && path[1] === "unlock") {
        return {
          ...state,
          spendContractId: path[2],
          selectedClauseIndex: 0,
          showUnlockInputErrors: false,
          error: undefined
        }
      }
      return {
        ...state,
        showUnlockInputErrors: false,
        error: undefined
      }
    default:
      return state
  }
}
