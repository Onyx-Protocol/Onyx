import Container from 'features/container/components/Container'
import prepareApplication from 'features/container/prepareApplication'
import { routes as authn } from 'features/authn'
import { routes as configuration } from 'features/configuration'
import appRoutes from 'features/app/appRoutes'

const makeRoutes = (store) => {
  return({
    path: '/',
    component: Container,
    onEnter: prepareApplication(store),
    childRoutes: [
      authn(store),
      configuration,
      appRoutes(store),
    ]
  })
}
export default makeRoutes
