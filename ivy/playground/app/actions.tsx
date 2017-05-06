import * as React from 'react'

import accounts from '../accounts'
import assets from '../assets'
import templates from '../templates'
import { selectTemplate } from '../contracts/actions'
import { RESET } from './constants'

export const reset = (dispatch, getState) => {
  return templates.actions.compileTemplates()(dispatch, getState).then((res) => {
    dispatch({ type: RESET })
    dispatch(selectTemplate("TrivialLock"))
    dispatch(accounts.actions.fetch())
    dispatch(assets.actions.fetch())
  })
}
