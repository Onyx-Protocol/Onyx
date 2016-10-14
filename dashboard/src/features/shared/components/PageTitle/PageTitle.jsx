import React from 'react'
import { connect } from 'react-redux'
import { Link } from 'react-router'
import { humanize } from 'utility/string'
import makeRoutes from 'routes'
import styles from './PageTitle.scss'

class PageTitle extends React.Component {
  render() {
    const chevron = require('assets/images/chevron.png')

    return(
      <div className={styles.main}>
        <div className={styles.navigation}>
          <ul className={styles.crumbs}>
            {this.props.breadcrumbs.map(crumb =>
              <li className={styles.crumb} key={crumb.name}>
                {!crumb.last && <Link to={crumb.path}>
                  {crumb.name}
                  <img src={chevron} className={styles.chevron} />
                </Link>}

                {crumb.last && <span className={styles.title}>
                  {this.props.title || crumb.name}
                </span>}
              </li>
            )}
          </ul>
        </div>

        {Array.isArray(this.props.actions) && <ul className={styles.actions}>
          {this.props.actions.map(item => <li key={item.key}>{item}</li>)}
        </ul>}
      </div>
    )
  }
}

const mapStateToProps = (state) => {
  const routes = makeRoutes()
  const pathname = state.routing.locationBeforeTransitions.pathname
  const breadcrumbs = []

  let currentRoutes = routes.childRoutes
  let currentPath = []
  pathname.split('/').forEach((component, index, array) => {
    let match = currentRoutes.find(route => {
      return route.path == component || route.path.indexOf(':') >= 0
    })

    if (match) {
      currentRoutes = match.childRoutes || []
      currentPath.push(component)

      if (!match.skipBreadcrumb) {
        breadcrumbs.push({
          last: (index == array.length -1),
          name: match.name || humanize(component),
          path: currentPath.join('/')
        })
      }
    }
  })

  return {
    breadcrumbs
  }
}

export default connect(
  mapStateToProps
)(PageTitle)
