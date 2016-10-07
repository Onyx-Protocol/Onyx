import { RoutingContainer } from 'features/shared/components'
import { List } from './components'

export default {
  path: 'unspents',
  component: RoutingContainer,
  indexRoute: { component: List }
}
