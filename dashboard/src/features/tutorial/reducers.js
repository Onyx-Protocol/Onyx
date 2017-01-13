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

export default combineReducers({
  step,
  isShowing,
})
