import Container from './components/Container'

import { getAuthenticationStatus } from 'features/authn/actions'
import { fetchCoreInfo } from 'features/core/actions'
import { fetchTestnetInfo } from 'features/testnet/actions'

const CORE_POLLING_TIME = 2 * 1000
const TESTNET_INFO_POLLING_TIME = 30 * 1000

export default (store) => ({
  path: '/',
  component: Container,
  onEnter: () => {
    store.dispatch(fetchTestnetInfo())
    store.dispatch(getAuthenticationStatus()).then(() => {
      setInterval(() => store.dispatch(fetchCoreInfo()), CORE_POLLING_TIME)
      setInterval(() => store.dispatch(fetchTestnetInfo()), TESTNET_INFO_POLLING_TIME)
    })
  }
})
