import { combineReducers } from 'redux'

export const step = (state = 0, action) => {
  if (action.type == 'TUTORIAL_NEXT_STEP') return state + 1
  return state
}

export const isShowing = (state = true, action) => {
  if (action.type == 'TOGGLE_TUTORIAL') return !state
  return state
}

export default combineReducers({
  step,
  isShowing,
})
