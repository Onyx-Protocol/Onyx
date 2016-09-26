import Section from '../../containers/SectionContainer'
import List from '../../containers/Accounts/List'
import New from '../../containers/Accounts/New'
import Show from './components/Show'

export default {
  path: 'accounts',
  component: Section,
  indexRoute: { component: List },
  childRoutes: [
    {
      path: 'create',
      component: New
    },
    {
      path: ':id',
      component: Show
    }
  ]
}
