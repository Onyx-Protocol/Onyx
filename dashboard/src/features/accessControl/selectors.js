import { createSelector } from 'reselect'
import { policyOptions } from './constants'

export const getPolicyNames = createSelector(
  item => item.grants,
  grants => grants.map(
    grant => {
      let isProtected = grant.protected
      let policy = grant.policy

      const found = policyOptions.find(elem => elem.value == policy)
      let label = found ? found.label : policy
      if (isProtected) {
        label = label + ' (Protected)'
      }
      return label
    }
  )
)

export const guardType = (item) => item.guardType

export const isAccessToken = createSelector(
  guardType,
  type => type == 'access_token'
)
