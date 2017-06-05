import Bootstrap from 'features/bootstrap/components/Bootstrap'
import prepareApplication from 'features/bootstrap/prepareApplication'
import { routes as authn } from 'features/authn'
import { routes as configuration } from 'features/configuration'
import appRoutes from 'features/app/appRoutes'

const makeRoutes = (store) => {
  return({
    path: '/',
    component: Bootstrap,
    onEnter: prepareApplication(store),
    childRoutes: [
      authn(store),
      configuration,
      appRoutes(store),
    ]
  })
}
export default makeRoutes
