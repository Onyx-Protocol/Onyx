import Section from '../../containers/SectionContainer'
import List from '../../containers/Assets/List'
import New from '../../containers/Assets/New'
import Show from './components/Show'

export default {
  path: 'assets',
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
