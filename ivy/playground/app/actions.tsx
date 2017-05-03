import * as React from 'react'

import accounts from '../accounts'
import assets from '../assets'

import { RESET } from './constants'

export const reset = (dispatch, getState) => {
  dispatch({ type: RESET })
  dispatch(accounts.actions.fetch())
  dispatch(assets.actions.fetch())
}
