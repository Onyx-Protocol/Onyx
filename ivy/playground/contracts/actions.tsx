// external imports
import { push } from 'react-router-redux'

// ivy imports
import { getItemMap } from '../assets/selectors';
import { getItem } from '../accounts/selectors';
import { fetch } from '../accounts/actions';
import { setSource, displayError as displayCreateError } from '../templates/actions'
import {
  getSource,
  getContractValue,
  getInputMap,
  getContractArgs,
} from '../templates/selectors'

import { getPromisedInputMap } from '../inputs/data'

// internal imports
import {
  getSpendContract,
  getSpendContractId,
  getSelectedClauseIndex,
  getLockActions,
  getRequiredValueAction,
  getUnlockAction,
  getClauseWitnessComponents,
  getClauseMintimes,
  getClauseMaxtimes
} from './selectors';

import {
  client,
  prefixRoute,
  createLockingTx,
  createUnlockingTx
} from '../core'

import {
  Action,
  ControlWithAccount,
  ControlWithReceiver,
  DataWitness,
  KeyId,
  Receiver,
  SignatureWitness,
  SpendUnspentOutput,
  WitnessComponent
} from '../core/types'

export const DISPLAY_ERROR = 'contracts/DISPLAY_ERROR'

export const displayError = (error) => {
  return {
    type: DISPLAY_ERROR,
    error
  }
}

export const CREATE_CONTRACT = 'contracts/CREATE_CONTRACT'

export const create = () => {
  return (dispatch, getState) => {
    const state = getState()
    const inputMap = getInputMap(state)
    if (inputMap === undefined) throw "create should not have been called when inputMap is undefined"

    const source = getSource(state)
    const spendFromAccount = getContractValue(state)
    if (spendFromAccount === undefined) throw "spendFromAccount should not be undefined here"
    const assetId = spendFromAccount.assetId
    const amount = spendFromAccount.amount
    const promisedInputMap = getPromisedInputMap(inputMap)
    const promisedTemplate = promisedInputMap.then((inputMap) => {
      const args = getContractArgs(state, inputMap).map(param => {
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
      return client.ivy.compile({ contract: source, args: args })
    }).catch(err => {
      console.log(err)
      dispatch(displayCreateError(err))
    })

    const promisedUtxo = promisedTemplate.then(template => {
      const receiver: Receiver = {
        controlProgram: template.program,
        expiresAt: "2017-06-25T00:00:00.000Z" // TODO
      }
      const controlWithReceiver: ControlWithReceiver = {
        type: "controlWithReceiver",
        receiver,
        assetId,
        amount
      }
      const actions: Action[] = [spendFromAccount, controlWithReceiver]
      return createLockingTx(actions)
    }).catch(err => {
      console.log(err)
      dispatch(displayCreateError(err))
    })

    Promise.all([promisedInputMap, promisedTemplate, promisedUtxo]).then(([inputMap, template, utxo]) => {
      dispatch({
        type: CREATE_CONTRACT,
        controlProgram: template.program,
        source,
        template,
        inputMap,
        utxo
      })
      dispatch(fetch())
      dispatch(setSource(source))
      dispatch(push(prefixRoute('/unlock')))
    }).catch(err => {
      console.log(err)
      dispatch(displayCreateError(err))
    })
  }
}

export const SPEND_CONTRACT = "contracts/SPEND_CONTRACT"

export const spend = () => {
  return(dispatch, getState) => {
    const state = getState()
    const contract = getSpendContract(state)
    const outputId = contract.outputId
    const lockedValueAction: SpendUnspentOutput = {
      type: "spendUnspentOutput",
      outputId
    }
    const lockActions: Action[] = getLockActions(state)
    const actions: Action[] = [lockedValueAction, ...lockActions]

    const reqValueAction = getRequiredValueAction(state)
    if (reqValueAction !== undefined) {
      actions.push(reqValueAction)
    }
    const unlockAction = getUnlockAction(state)
    if (unlockAction !== undefined) {
      actions.push(unlockAction)
    }

    const witness: WitnessComponent[] = getClauseWitnessComponents(getState())
    const mintimes = getClauseMintimes(getState())
    const maxtimes = getClauseMaxtimes(getState())
    createUnlockingTx(actions, witness, mintimes, maxtimes).then((result) => {
      dispatch({
        type: SPEND_CONTRACT,
        id: contract.id,
        unlockTxid: result.id
      })
      dispatch(fetch())
      dispatch(push(prefixRoute('/unlock')))
    }).catch(err => {
      console.log(err)
      dispatch(displayError(err))
    })
  }
}

export const SET_CLAUSE_INDEX = 'contracts/SET_CLAUSE_INDEX'

export const setClauseIndex = (selectedClauseIndex: number) => {
  return {
    type: SET_CLAUSE_INDEX,
    selectedClauseIndex: selectedClauseIndex
  }
}

export const UPDATE_INPUT = 'contracts/UPDATE_INPUT'

export const updateInput = (name: string, newValue: string) => {
  return {
    type: UPDATE_INPUT,
    name: name,
    newValue: newValue
  }
}

export const UPDATE_CLAUSE_INPUT = 'contracts/UPDATE_CLAUSE_INPUT'

export const updateClauseInput = (name: string, newValue: string) => {
  return (dispatch, getState) => {
    const state = getState()
    const contractId = getSpendContractId(state)
    dispatch({
      type: UPDATE_CLAUSE_INPUT,
      contractId: contractId,
      name: name,
      newValue: newValue
    })
  }
}
