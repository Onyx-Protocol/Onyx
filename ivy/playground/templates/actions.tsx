import { client } from '../util'
import { getItemMap, getIdList, getTemplate } from './selectors'
import { Item as Template, ItemMap } from './types'
import { TRIVIAL_LOCK, LOCK_WITH_PUBLIC_KEY, LOCK_TO_OUTPUT, TRADE_OFFER, ESCROWED_TRANSFER } from './constants'
import { selectTemplate } from '../contracts/actions'
import { ContractParameter, TemplateClause, ClauseParameter } from 'ivy-compiler'

export const COMPILE_TEMPLATES = 'templates/COMPILE_TEMPLATES'
export const LOAD_TEMPLATE = 'templates/LOAD_TEMPLATE'
export const SET_SOURCE = 'templates/SET_SOURCE'

// export type TemplateClause = {
//   type: "templateClause",
//   name: string,
//   parameters: ClauseParameter[],
//   outputs: Output[],
//   returnStatement?: Return
// }
//
// export type Output = {
//   type: "output",
//   location: Location,
//   contract: ContractExpression,
//   assetAmountParam?: string,
//   index?: number
// }
//
// export type ContractExpression = {
//   type: "contractExpression",
//   location: Location,
//   address: Variable,
//   value: Variable | StoredValue
// }

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

export const compileTemplates = () => {
  return (dispatch, getState) => {
    const itemMap: ItemMap = {}
    return Promise.all([
      client.ivy.compile({ contract: TRIVIAL_LOCK }),
      client.ivy.compile({ contract: LOCK_WITH_PUBLIC_KEY }),
      client.ivy.compile({ contract: LOCK_TO_OUTPUT }),
      client.ivy.compile({ contract: TRADE_OFFER }),
      client.ivy.compile({ contract: ESCROWED_TRANSFER }),
    ]).then(result => {
      itemMap["TrivialLock"] = mapServerTemplate(result[0])
      itemMap["LockWithPublicKey"] = mapServerTemplate(result[1])
      itemMap["LockToOutput"] = mapServerTemplate(result[2])
      itemMap["TradeOffer"] = mapServerTemplate(result[3])
      itemMap["EscrowedTransfer"] = mapServerTemplate(result[4])
      const idList = ["TrivialLock", "LockWithPublicKey", "LockToOutput", "TradeOffer", "EscrowedTransfer"]
      const source = itemMap["TrivialLock"].source
      const selected = idList[0]
      dispatch({
        type: COMPILE_TEMPLATES,
        itemMap,
        idList,
        selected,
        source
      })
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
