import createHash from 'sha.js'

export default (state = {ids: [], items: {}}, action) => {
  // Grant list is always complete, so we rebuild state from scratch
  if (action.type == 'RECEIVED_ACCESS_GRANTS') {
    const newObjects = {}

    action.tokens.forEach(token => {
      const tokenGuard = {
        id: token.id
      }
      const id = createHash('sha256').update(JSON.stringify(tokenGuard), 'utf8').digest('hex')
      newObjects[id] = {
        id: id,
        name: token.id,
        guardType: 'access_token',
        guardData: tokenGuard,
        policies: [],
        latestGrant: '0000'
      }
    })

    action.grants.forEach(grant => {
      const id = createHash('sha256').update(JSON.stringify(grant.guardData), 'utf8').digest('hex')

      if (newObjects[id]) {
        newObjects[id].policies.push(grant.policy)
        if (newObjects[id].latestGrant.localeCompare(grant.createdAt) < 0) {
          newObjects[id].latestGrant = grant.createdAt
        }
      } else {
        newObjects[id] = {
          id: id,
          guardType: grant.guardType,
          guardData: grant.guardData,
          policies: [grant.policy],
          latestGrant: grant.createdAt
        }
      }
    })

    const newIds = Object.values(newObjects)
      .sort((a, b) => b.latestGrant.localeCompare(a.latestGrant))
      .map(object => object.id)

    return {
      ids: newIds,
      items: newObjects
    }
  } else if (action.type == 'DELETE_ACCESS_TOKEN') {
    const ids = [...state.ids]
    const items = {...state.items}

    const idToRemove = action.id
    const deleteIndex = ids.indexOf(idToRemove)
    ids.splice(deleteIndex, 1)

    delete items[idToRemove]

    return {
      ids,
      items
    }
  }

  return state
}
