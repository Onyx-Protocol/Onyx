import { mapServerTemplate } from '../templates/util';
import { AssetAliasInput, ProgramInput } from '../inputs/types';
import { getItemMap } from '../assets/selectors';
import { getItem } from '../accounts/selectors';
export const CREATE_CONTRACT = 'contracts/CREATE_CONTRACT'
export const UPDATE_INPUT = 'contracts/UPDATE_INPUT'
import { push } from 'react-router-redux'
import {
  getClauseParameterIds,
  getClauseDataParameterIds,
  getSpendContractId,
  getClauseWitnessComponents,
  getSpendContractSelectedClauseIndex,
  getClauseOutputActions,
  getClauseValue,
  getClauseReturnAction,
  getClauseMintimes,
  getClauseMaxtimes
} from './selectors';

import {
  getSource,
  getContractValue,
  getInputMap,
  getParameterData,
} from '../templates/selectors'

import { getPromisedInputMap } from '../inputs/data'

import {
  client,
  prefixRoute,
  createLockingTx,
  createUnlockingTx
} from '../core'

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
} from '../core/types'


export const SELECT_TEMPLATE = 'contracts/SELECT_TEMPLATE'
export const SET_CLAUSE_INDEX = 'contracts/SET_CLAUSE_INDEX'
export const SPEND = 'contracts/SPEND'
export const SHOW_ERRORS = 'contracts/SHOW_ERRORS'

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
    if (inputMap === undefined) throw "create should not have been called when inputMap is undefined"
    let promisedInputMap = getPromisedInputMap(inputMap)
    promisedInputMap.then((inputMap) => {
      const args = getParameterData(state, inputMap).map(param => {
        if (param instanceof Buffer) {
          return { "string": param.toString('hex') }
        }

        if (typeof param === 'string') {
          return { "string": param }
        }

        if (typeof param === 'number') {
          return { "integer": param }
        }

        if (typeof param === 'boolean') {
          return { 'boolean': param }
        }
        throw 'unsupported argument type ' + (typeof param)
      })
      const source = getSource(state)
      client.ivy.compile({ contract: source, args: args }).then(contract => {
        let controlProgram = contract.program
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
        let template = mapServerTemplate(contract)
        let actions: Action[] = [spendFromAccount, controlWithReceiver]
        return createLockingTx(actions).then(utxo => {
          dispatch({
            type: CREATE_CONTRACT,
            controlProgram,
            source,
            template,
            inputMap,
            utxo
          })
          dispatch(push(prefixRoute('/unlock')))
        })
      })
    }).catch(err => {
      console.log("error found", err)
    })
  }
}

export const SPEND_CONTRACT = "contracts/SPEND_CONTRACT"

export const spend = () => {
  return(dispatch, getState) => {
    const state = getState()
    const contract = getSpendContract(state)
    const clauseIndex = getSpendContractSelectedClauseIndex(state)
    const outputId = contract.outputId
    const spendContractAction: SpendUnspentOutput = {
      type: "spendUnspentOutput",
      outputId
    }

    const clauseOutputActions: Action[] = getClauseOutputActions(state)
    const actions: Action[] = [spendContractAction, ...clauseOutputActions]
    const clauseValue = getClauseValue(state)
    if (clauseValue !== undefined) {
      actions.push(clauseValue)
    }
    const returnAction = getClauseReturnAction(state)
    if (returnAction !== undefined) {
      actions.push(returnAction)
    }

    const clauseParams = getClauseParameterIds(state)
    const clauseDataParams = getClauseDataParameterIds(state)
    const witness: WitnessComponent[] = getClauseWitnessComponents(getState())
    const mintimes = getClauseMintimes(getState())
    const maxtimes = getClauseMaxtimes(getState())
    createUnlockingTx(actions, witness, mintimes, maxtimes).then((result) => {
      dispatch({
        type: SPEND_CONTRACT,
        id: contract.id,
        lockTxid: result.id
      })
      dispatch(push(prefixRoute('/unlock')))
    })
  }
}

export const setClauseIndex = (selectedClauseIndex: number) => {
  return {
    type: SET_CLAUSE_INDEX,
    selectedClauseIndex: selectedClauseIndex
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
