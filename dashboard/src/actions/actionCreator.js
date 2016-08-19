const actionCreator = function(type, payloadCreator) {
  const res = (...args) => {
    let action = { type }

    if (typeof payloadCreator == "function") {
      return Object.assign({}, payloadCreator(...args), action)
    }

    return action
  }
  res.type = type

  return res
}

export default actionCreator
