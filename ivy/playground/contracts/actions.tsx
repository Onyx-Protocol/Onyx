import { AssetAliasInput, ProgramInput } from '../inputs/types';
import { getItemMap } from '../assets/selectors';
import { getItem } from '../accounts/selectors';
export const CREATE_CONTRACT = 'contracts/CREATE_CONTRACT'
export const UPDATE_INPUT = 'contracts/UPDATE_INPUT'
import { push } from 'react-router-redux'
import {
  getClauseParameterIds,
  getClauseDataParameterIds,
  getInputMap,
  getControlProgram,
  getContractValue,
  getSelectedTemplate,
  getSpendContractId,
  getClauseWitnessComponents,
  getSpendContractSelectedClauseIndex,
  getClauseOutputActions
} from './selectors';

import { getPromisedInputMap } from '../inputs/data'

import {
  WitnessComponent,
  KeyId,
  DataWitness,
  SignatureWitness,
  Receiver,
  SpendUnspentOutput,
  ControlWithAccount,
  ControlWithReceiver,
  Action
} from '../transactions/types'
import { createFundingTx, createSpendingTx } from '../transactions'
import { prefixRoute } from '../util'

export const SELECT_TEMPLATE = 'contracts/SELECT_TEMPLATE'
export const SET_CLAUSE_INDEX = 'contracts/SET_CLAUSE_INDEX'
export const SPEND = 'contracts/SPEND'
export const SHOW_ERRORS = 'contracts/SHOW_ERRORS'

import { getItemMap as getTemplateMap } from '../templates/selectors'
import { getSpendContract } from './selectors'

import { InputMap } from '../inputs/types'

export const showErrors = () => {
  return {
    type: SHOW_ERRORS
  }
}

export const create = () => {
  return (dispatch, getState) => {
    let state = getState()
    let inputMap = getInputMap(state)
    let promisedInputMap = getPromisedInputMap(inputMap)
    promisedInputMap.then((inputMap) => {
      let controlProgram = getControlProgram(state, inputMap)
      if (controlProgram === undefined) return
      let spendFromAccount = getContractValue(state)
      if (spendFromAccount === undefined) throw "spendFromAccount should not be undefined here"
      let assetId = spendFromAccount.assetId
      let amount = spendFromAccount.amount
      let receiver: Receiver = {
        controlProgram: controlProgram,
        expiresAt: "2017-06-25T00:00:00.000Z" // TODO
      }
      let controlWithReceiver: ControlWithReceiver = {
        type: "controlWithReceiver",
        receiver,
        assetId,
        amount
      }
      let actions: Action[] = [spendFromAccount, controlWithReceiver]
      return createFundingTx(actions).then(utxo => {
        dispatch({
          type: CREATE_CONTRACT,
          controlProgram: controlProgram,
          template: getSelectedTemplate(state),
          inputMap: inputMap,
          utxo: utxo
        })
        dispatch(push(prefixRoute('/spend')))
      })
    }).catch(err => {
      dispatch(showErrors())
    })
  }
}

export const spend = () => {
  return(dispatch, getState) => {
    let state = getState()
    let contract = getSpendContract(state)
    let clauseIndex = getSpendContractSelectedClauseIndex(state)
    let outputId = contract.outputId
    let spendInputMap = contract.spendInputMap
    let spendContractAction: SpendUnspentOutput = {
      type: "spendUnspentOutput",
      outputId
    }
    let clauseOutputActions: Action[] = getClauseOutputActions(state)
    let actions: Action[] = [spendContractAction, ...clauseOutputActions]
    let returnInput = spendInputMap["transactionDetails.accountAliasInput"]
    if (returnInput !== undefined) {
      actions.push({
        type: "controlWithAccount",
        accountId: returnInput.value,
        assetId: contract.assetId,
        amount: contract.amount
      } as ControlWithAccount)
    }
    let clauseParams = getClauseParameterIds(state)
    let clauseDataParams = getClauseDataParameterIds(state)
    console.log("actions", actions)
    const witness: WitnessComponent[] = getClauseWitnessComponents(getState())
    createSpendingTx(actions, witness).then((result) => {
      console.log("result", result)
    })
    // dispatch({
    //   type: SPEND
    // })
  }
}

export const setClauseIndex = (selectedClauseIndex: number) => {
  return {
    type: SET_CLAUSE_INDEX,
    selectedClauseIndex: selectedClauseIndex
  }
}

export const selectTemplate = (templateId: string) => {
  return(dispatch, getState) => {
    let templateMap = getTemplateMap(getState())
    dispatch({
      type: SELECT_TEMPLATE,
      template: templateMap[templateId],
      templateId
    })
  }
}

export function updateInput(name: string, newValue: string) {
  return {
    type: UPDATE_INPUT,
    name: name,
    newValue: newValue
  }
}

export const UPDATE_CLAUSE_INPUT = 'UPDATE_CLAUSE_INPUT'

export function updateClauseInput(name: string, newValue: string) {
  return (dispatch, getState) => {
    let state = getState()
    let contractId = getSpendContractId(state)
    dispatch({
      type: UPDATE_CLAUSE_INPUT,
      contractId: contractId,
      name: name,
      newValue: newValue
    })
  }
}
