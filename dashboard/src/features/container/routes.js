import Container from './components/Container'
import actions from 'actions'

const CORE_POLLING_TIME = 2 * 1000
const TESTNET_INFO_POLLING_TIME = 30 * 1000

export default (store) => ({
  path: '/',
  component: Container,
  onEnter: () => {
    store.dispatch(actions.core.fetchCoreInfo())
    store.dispatch(actions.testnet.fetchTestnetInfo())

    setInterval(() => store.dispatch(actions.core.fetchCoreInfo()), CORE_POLLING_TIME)
    setInterval(() => store.dispatch(actions.testnet.fetchTestnetInfo()), TESTNET_INFO_POLLING_TIME)
  }
})
