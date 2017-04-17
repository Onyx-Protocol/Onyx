import { createSelector } from 'reselect'
import { policyOptions } from './constants'

export const getPolicyNames = createSelector(
  item => item.policies,
  policies => policies.map(
    policy => policyOptions.find(
      elem => elem.value == policy
    ).label
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
