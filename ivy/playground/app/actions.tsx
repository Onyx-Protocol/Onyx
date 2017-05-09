import * as React from 'react'

import accounts from '../accounts'
import assets from '../assets'
import templates from '../templates'
import { selectTemplate } from '../contracts/actions'
import { load } from '../templates/actions'
import { RESET } from './constants'

export const reset = (dispatch, getState) => {
  dispatch({ type: RESET })
  dispatch(selectTemplate("TrivialLock"))
  dispatch(load("TrivialLock"))
  dispatch(accounts.actions.fetch())
  dispatch(assets.actions.fetch())
}
