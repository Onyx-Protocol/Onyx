import actions from '../actions'

export const pagesActions = (type) => (state = [], action) => {
  if (action.type == actions[type].resetPage.type) {
    return []
  } else if (action.type == actions[type].appendPage.type) {
    return state.concat([action.param])
  }

  return state
}

export const currentPageActions = (type) => (state = -1, action) => {
  if (action.type == actions[type].resetPage.type) {
    return -1
  } else if (action.type == actions[type].incrementPage.type) {
    return state + 1
  } else if (action.type == actions[type].decrementPage.type) {
    return Math.max(state - 1, 0)
  }

  return state
}

export const currentQueryActions = (type) => (state = "", action) => {
  if (action.type == actions[type].updateQuery.type) {
    return action.param
  }

  return state
}
