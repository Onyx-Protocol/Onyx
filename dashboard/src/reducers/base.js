import actions from '../actions'

export const pagesActions = (type) => (state = [], action) => {
  if ((actions[type].created && action.type == actions[type].created.type) ||
    action.type == actions[type].updateQuery.type) {
    return []
  } else if (action.type == actions[type].appendPage.type) {
    return state.concat([action.param])
  }

  return state
}

export const currentPageActions = (type) => (state = -1, action) => {
  if ((actions[type].created && action.type == actions[type].created.type) ||
    action.type == actions[type].updateQuery.type) {
    return -1
  } else if (action.type == actions[type].appendPage.type ||
    action.type == actions[type].incrementPage.type) {
    return state + 1
  } else if (action.type == actions[type].decrementPage.type) {
    return Math.max(state - 1, 0)
  }

  return state
}

export const currentQueryActions = (type) => (state = "", action) => {
  if (action.type == actions[type].updateQuery.type) {
    if (action.param && action.param.query) {
      return action.param.query
    } else if (typeof action.param === "string") {
      return action.param
    }

    return ""
  } else if (actions[type].created && action.type == actions[type].created.type) {
    return ""
  }

  return state
}
