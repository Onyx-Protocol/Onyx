import AccessControlList from './components/AccessControlList'
import NewToken from './components/NewToken'
import NewCertificate from './components/NewCertificate'
import { makeRoutes } from 'features/shared'
import actions from './actions'

const checkParams = (nextState, replace) => {
  if (!['token', 'certificate'].includes(nextState.location.query.type)) {
    replace({
      pathname: '/access-control',
      search: '?type=token',
      state: {preserveFlash: true}
    })
    return false
  }
  return true
}

export default (store) => {
  const loadGrants = () => store.dispatch(actions.fetchItems())

  const routes = makeRoutes(store, 'accessControl', AccessControlList, null, null, null, {
    path: 'access-control',
    name: 'Access control'
  })

  routes.indexRoute.onEnter = (nextState, replace) => {
    if (checkParams(nextState, replace)) { loadGrants() }
  }

  routes.indexRoute.onChange = (_, nextState, replace) => {
    checkParams(nextState, replace)
  }

  routes.childRoutes.push({
    path: 'create-token',
    component: NewToken
  })

  routes.childRoutes.push({
    path: 'add-certificate',
    component: NewCertificate
  })

  return routes
}
