import steps from './steps.json'

export const step = (state = 0, action) => {
  if (action.type == 'TUTORIAL_NEXT_STEP') return state + 1
  if (action.type == 'UPDATE_TUTORIAL' && steps[state].objectType == action.object) {
    return state + 1
  }
  if (action.type == 'DISMISS_TUTORIAL') return 0
  return state
}

export const isShowing = (state = process.env.NODE_ENV != 'test', action) => {
  if (action.type == 'DISMISS_TUTORIAL') return false
  if (action.type == 'OPEN_TUTORIAL') return true
  return state
}

export const route = (currentStep) => (state = '/', action) => {
  if (action.type == 'TUTORIAL_NEXT_STEP') return action.route
  if (action.type == 'UPDATE_TUTORIAL' && currentStep.objectType == action.object) {
    return action.object + 's'
  }
  if (action.type == 'DISMISS_TUTORIAL') return '/'
  return state
}

export const userInputs = (currentStep) => (state = { accounts: [] }, action) => {
  if (action.type == 'UPDATE_TUTORIAL' && currentStep.objectType == action.object) {
    if (action.object == 'mockhsm') return { ...state, mockhsm: action.data }
    if (action.object == 'asset') return { ...state, asset: action.data }
    if (action.object == 'account') {
      return { ...state, accounts: [...state.accounts, action.data] }
    }
    return state
  }
  if (action.type == 'DISMISS_TUTORIAL') return { accounts: [] }
  return state
}

export default (state = {}, action) => {
  const newState = {
    step: step(state.step, action),
    isShowing: isShowing(state.isShowing, action)
  }

  newState.currentStep = steps[newState.step]
  newState.userInputs = userInputs(newState.currentStep)(state.userInputs, action)
  newState.route = route(newState.currentStep)(state.route, action)

  return newState
}
