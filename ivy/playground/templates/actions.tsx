import { client } from '../util'
import { getItemMap, getIdList, getTemplate } from './selectors'
import { Item as Template, ItemMap } from './types'
import { TRIVIAL_LOCK, LOCK_WITH_PUBLIC_KEY, LOCK_TO_OUTPUT, TRADE_OFFER, ESCROWED_TRANSFER,
         COLLATERALIZED_LOAN } from './constants'
import { selectTemplate } from '../contracts/actions'
import { ContractParameter, TemplateClause, ClauseParameter } from 'ivy-compiler'

export const SET_INITIAL_TEMPLATES = 'templates/SET_TEMPLATES'
export const LOAD_TEMPLATE = 'templates/LOAD_TEMPLATE'
export const SET_SOURCE = 'templates/SET_SOURCE'

const mapServerTemplate = (tpl): Template => {
  const clauses: TemplateClause[] = tpl.clauseInfo.map(clause => {
    const parameters: ClauseParameter[] = clause.args.map(param => ({
      type: "clauseParameter",
      valueType: param.type,
      identifier: param.name
    } as ClauseParameter))

    let returnStatement
    let outputs = clause.valueInfo.filter(value => {
      if (value.program === undefined) {
        // return statement
        // TODO(boymanjor): detect and handle variable return statements
        returnStatement = {
          type: "returnStatement",
          value: {
            type: "storedValue",
            identifier: value.name
          }
        }
        return false
      }
      return true
    })

    outputs = outputs.map((output, idx) => {
      return {
        type: "output",
        contract: {
          type: "contractExpression",
          address: {
            type: "variable",
            identifier: output.program
          },
          value: {
            type: "storedValue",
            identifier: output.name
          },
        },
        assetAmountParam: output.assetAmount,
        index: idx
      }
    })

    return {
      type: "templateClause",
      name: clause.name,
      parameters,
      outputs,
      returnStatement
    } as TemplateClause
  })

  const contractParameters: ContractParameter[] = tpl.params.map(param => ({
    type: "contractParameter",
    valueType: param.type,
    identifier: param.name
  }) as ContractParameter)

  return {
    name: tpl.name,
    instructions: tpl.opcodes.split(" "),
    source: tpl.source,
    contractParameters,
    clauses
  } as Template
}

export const setInitialTemplates = () => {
  return (dispatch, getState) => {
    Promise.all([
      client.ivy.compile({ contract: TRIVIAL_LOCK }),
      client.ivy.compile({ contract: LOCK_WITH_PUBLIC_KEY }),
      client.ivy.compile({ contract: LOCK_TO_OUTPUT }),
      client.ivy.compile({ contract: TRADE_OFFER }),
      client.ivy.compile({ contract: ESCROWED_TRANSFER }),
      client.ivy.compile({ contract: COLLATERALIZED_LOAN }),
    ]).then(result => {
      const itemMap = {
        TrivialLock: mapServerTemplate(result[0]),
        LockWithPublicKey: mapServerTemplate(result[1]),
        LockToOutput: mapServerTemplate(result[2]),
        TradeOffer: mapServerTemplate(result[3]),
        EscrowedTransfer: mapServerTemplate(result[4]),
        CollateralizedLoan: mapServerTemplate(result[5])
      }
      const idList = [
        "TrivialLock",
        "LockWithPublicKey",
        "LockToOutput",
        "TradeOffer",
        "EscrowedTransfer",
        "CollateralizedLoan"
      ]
      const selected = idList[0]
      const source = itemMap[selected].source
      dispatch({
        type: SET_INITIAL_TEMPLATES,
        itemMap,
        idList,
        source,
        selected
      })
      dispatch(selectTemplate(selected))
    }).catch(err => {
      throw err
    })
  }
}

export const load = (selected: string) => {
  return {
    type: LOAD_TEMPLATE,
    selected: selected
  }
}

export const setSource = (source: string) => {
  return (dispatch, getState) => {
    const type = SET_SOURCE
    return dispatch({ type, source })
  }
}

export const SAVE_TEMPLATE = 'SAVE_TEMPLATE'

export function saveTemplate() {
  return (dispatch, getState) => {
    let template = getTemplate(getState())
    dispatch({
      type: SAVE_TEMPLATE,
      template: template
    })
  }
}
