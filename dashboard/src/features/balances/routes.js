import { RoutingContainer } from 'features/shared/components'
import { List } from 'features/balances/components'

export default {
  path: 'balances',
  component: RoutingContainer,
  indexRoute: { component: List },
}
