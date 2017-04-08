import { RoutingContainer } from 'features/shared/components'
import { humanize } from 'utility/string'
import actions from 'actions'

const makeRoutes = (store, type, List, New, Show, Update, options = {}) => {
  const loadPage = (state, replace) => {
    const query = state.location.query
    if (query.filter && options.skipFilter) {
      replace(state.location.pathname)
      return
    } else if (query.filter === undefined && options.defaultFilter) {
      replace(`${state.location.pathname}?filter=${options.defaultFilter}`)
      return
    }

    const pageNumber = parseInt(state.location.query.page || 1)
    if (pageNumber == 1) {
      store.dispatch(actions[type].fetchPage(query, pageNumber, { refresh: true }))
    } else {
      store.dispatch(actions[type].fetchPage(query, pageNumber))
    }
  }

  const childRoutes = []

  if (New) {
    childRoutes.push({
      path: 'create',
      component: New
    })
  }

  if (options.childRoutes) {
    childRoutes.push(...options.childRoutes)
  }

  if (Show) {
    childRoutes.push({
      path: ':id',
      component: Show
    })
  }

  if (Update) {
    childRoutes.push({
      path: ':id/tags',
      component: Update
    })
  }

  return {
    path: options.path || type + 's',
    component: RoutingContainer,
    name: options.name || humanize(type + 's'),
    indexRoute: {
      component: List,
      onEnter: (nextState, replace) => { loadPage(nextState, replace) },
      onChange: (_, nextState, replace) => { loadPage(nextState, replace) }
    },
    childRoutes: childRoutes
  }
}

export default makeRoutes
