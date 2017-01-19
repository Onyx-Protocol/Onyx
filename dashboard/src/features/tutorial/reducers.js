import { combineReducers } from 'redux'

export const step = (state = 0, action) => {
  if (action.type == 'TUTORIAL_NEXT_STEP') return state + 1
  else if (action.type == 'DISMISS_TUTORIAL') return 0
  return state
}

export const isShowing = (state = true, action) => {
  if (action.type == 'DISMISS_TUTORIAL') return false
  else if (action.type == 'OPEN_TUTORIAL') return true
  return state
}

export const userInputs = (state = { accounts: [] }, action) => {
  if (action.type == 'UPDATE_TUTORIAL'){
    if (action.object == 'mockhsm') {
      return {...state, mockhsm: action.data}
    }
    else if (action.object == 'asset') {
      return {...state, asset: action.data}
    }
    else if (action.object == 'account') {
      return {...state, accounts: [...state.accounts, action.data] }
    }
    return state
  }
  else if (action.type == 'DISMISS_TUTORIAL'){
    return { accounts: [] }
  }
  return state
}

export default combineReducers({
  step,
  isShowing,
  userInputs
})
