import { createSelector } from 'reselect'
import { policyOptions, protectedSuffix } from './constants'

export const getPolicyNames = createSelector(
  item => item.policies,
  policies => policies.map(
    policy => {
      let isProtected = false
      if (policy.indexOf(protectedSuffix) >= 0) {
        policy = policy.replace(protectedSuffix, '')
        isProtected = true
      }

      const found = policyOptions.find(elem => elem.value == policy)
      let label = found ? found.label : policy
      if (isProtected) {
        label = label + ' (Protected)'
      }
      return label
    }
  )
)

export const getPolicyNamesString = createSelector(
  getPolicyNames,
  names => names.join(', ')
)

export const guardType = (item) => item.guardType

export const isAccessToken = createSelector(
  guardType,
  type => type == 'access_token'
)
