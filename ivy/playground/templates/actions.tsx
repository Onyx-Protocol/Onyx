import { client } from '../util'
import { getItemMap, getIdList, getTemplate } from './selectors'
import { ItemMap } from './types'
import { TRIVIAL_LOCK, LOCK_WITH_PUBLIC_KEY, LOCK_TO_OUTPUT, TRADE_OFFER, ESCROWED_TRANSFER } from './constants'

export const COMPILE_TEMPLATES = 'templates/COMPILE_TEMPLATES'
export const LOAD_TEMPLATE = 'templates/LOAD_TEMPLATE'
export const SET_SOURCE = 'templates/SET_SOURCE'

export const compileTemplates = () => {
  return (dispatch, getState) => {
    const itemMap: ItemMap = {}
    Promise.all([
      client.ivy.compile({ contract: TRIVIAL_LOCK }),
      client.ivy.compile({ contract: LOCK_WITH_PUBLIC_KEY }),
      client.ivy.compile({ contract: LOCK_TO_OUTPUT }),
      client.ivy.compile({ contract: TRADE_OFFER }),
      client.ivy.compile({ contract: ESCROWED_TRANSFER }),
    ]).then(result => {
      console.log(result)
      itemMap["TrivialLock"] = result[0]
      itemMap["LockWithPublicKey"] = result[1]
      itemMap["LockToOutput"] = result[2]
      itemMap["TradeOffer"] = result[3]
      itemMap["EscrowedTransfer"] = result[4]
      const idList = ["TrivialLock", "LockWithPublicKey", "LockToOutput", "TradeOffer", "EscrowedTransfer"]
      const source = itemMap["TrivialLock"].source
      const selected = idList[0]
      dispatch({
        type: COMPILE_TEMPLATES,
        itemMap
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
