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
        grants: [],
        createdAt: token.createdAt
      }
    })

    action.grants.forEach(grant => {
      const id = createHash('sha256').update(JSON.stringify(grant.guardData), 'utf8').digest('hex')

      if (newObjects[id]) {
        const existingIndex = newObjects[id].grants.findIndex(g => g.policy == grant.policy)
        if (existingIndex >= 0) {
          const existing = newObjects[id].grants[existingIndex]
          if (existing.protected) { return }
          if (grant.protected) { newObjects[id].grants.splice(existingIndex, 1) }
        }

        newObjects[id].grants.push(grant)
        if (newObjects[id].createdAt.localeCompare(grant.createdAt) > 0) {
          newObjects[id].createdAt = grant.createdAt
        }
      } else {
        newObjects[id] = {
          id: id,
          guardType: grant.guardType,
          guardData: grant.guardData,
          grants: [grant],
          createdAt: grant.createdAt
        }
      }
    })

    const newIds = Object.values(newObjects)
      .sort((a, b) => b.createdAt.localeCompare(a.createdAt))
      .map(object => object.id)

    return {
      ids: newIds,
      items: newObjects
    }
  } else if (action.type == 'BEGIN_POLICY_EDITING') {
    const id = action.id
    const item = {...state.items[id]}
    item.isEditing = true

    return {
      ids: state.ids,
      items: {
        ...state.items,
        [id]: item
      }
    }
  } else if (action.type == 'END_POLICY_EDITING') {
    const id = action.id
    const item = {...state.items[id]}
    item.isEditing = false
    if (action.policies) {
      item.grants = Object.keys(action.policies)
        .filter(policy => action.policies[policy])
        .map(policy => ({
          ...item.grants.find(grant => grant.policy == policy),
          policy: policy
        }))
    }

    return {
      ids: state.ids,
      items: {
        ...state.items,
        [id]: item
      }
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
