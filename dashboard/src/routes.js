import Bootstrap from 'features/bootstrap/components/Bootstrap'
import prepareApplication from 'features/bootstrap/prepareApplication'
import authnRoutes from 'features/authn/routes'
import configurationRoutes from 'features/configuration/routes'
import appRoutes from 'features/app/appRoutes'

const makeRoutes = (store) => {
  return({
    path: '/',
    component: Bootstrap,
    onEnter: prepareApplication(store),
    childRoutes: [
      authnRoutes(store),
      configurationRoutes,
      appRoutes(store),
    ]
  })
}
export default makeRoutes
