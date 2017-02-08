import { combineReducers } from 'redux'
import steps from './steps.json'

export const step = (state = 0, action) => {
  if (action.type == 'TUTORIAL_NEXT_STEP'){
    return state + 1
  }
  else if (action.type == 'UPDATE_TUTORIAL' && steps[state].objectType == action.object) {
    return state + 1
  }
  else if (action.type == 'DISMISS_TUTORIAL') return 0
  return state
}

export const isShowing = (state = true, action) => {
  if (action.type == 'DISMISS_TUTORIAL') return false
  else if (action.type == 'OPEN_TUTORIAL') return true
  return state
}

export const route = (currentStep) => (state = 'transactions', action) => {
  if (action.type == 'TUTORIAL_NEXT_STEP'){
    return action.route
  }
  else if (action.type == 'UPDATE_TUTORIAL' && currentStep.objectType == action.object){
    return action.object + 's'
  }
  else if (action.type == 'DISMISS_TUTORIAL'){
    return 'transactions'
  }
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

export default (state = {}, action) => {
  const tutorialRoute = state.route
  delete state.currentStep // combineReducers logs error because currentStep set outside of function
  delete state.route
  const newState = combineReducers({
    step,
    isShowing,
    userInputs
  })(state, action)
  newState.currentStep = steps[newState.step]
  newState.route = route(steps[newState.step - 1])(tutorialRoute, action)
  return newState
}
