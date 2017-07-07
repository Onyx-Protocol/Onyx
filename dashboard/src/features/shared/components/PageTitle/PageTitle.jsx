import React from 'react'
import { connect } from 'react-redux'
import { Flash } from 'features/shared/components'
import { Link } from 'react-router'
import { humanize, capitalize } from 'utility/string'
import makeRoutes from 'routes'
import styles from './PageTitle.scss'
import componentClassNames from 'utility/componentClassNames'

class PageTitle extends React.Component {
  render() {
    const chevron = require('images/chevron.png')

    return(
      <div className={componentClassNames(this)}>
        <div className={styles.main}>
          <div className={styles.navigation}>
            <ul className={styles.crumbs}>
              {this.props.breadcrumbs.map(crumb =>
                <li className={styles.crumb} key={crumb.name}>
                  {!crumb.last && <Link to={crumb.path}>
                    {capitalize(crumb.name)}
                    <img src={chevron} className={styles.chevron} />
                  </Link>}

                  {crumb.last && <span className={styles.title}>
                    {this.props.title || crumb.name}
                  </span>}
                </li>
              )}
            </ul>
          </div>

          {!this.props.hideActions && Array.isArray(this.props.actions) && <ul className={styles.actions}>
            {this.props.actions.map(item => <li key={item.key}>{item}</li>)}
          </ul>}
        </div>

        <Flash messages={this.props.flashMessages}
          markFlashDisplayed={this.props.markFlashDisplayed}
          dismissFlash={this.props.dismissFlash}
        />
      </div>
    )
  }
}

const mapStateToProps = (state) => {
  const breadcrumbRoot = makeRoutes().childRoutes.find(route => route.useForBreadcrumbs)
  const pathname = state.routing.locationBeforeTransitions.pathname
  const breadcrumbs = []

  let currentRoutes = breadcrumbRoot.childRoutes
  let currentPath = []
  pathname.split('/').forEach(component => {
    let match = currentRoutes.find(route => {
      return route.path == component || route.path.indexOf(':') >= 0
    })

    if (match) {
      currentRoutes = match.childRoutes || []
      currentPath.push(component)

      if (!match.skipBreadcrumb) {
        breadcrumbs.push({
          name: match.name || humanize(component),
          path: `/${currentPath.join('/')}`
        })
      }
    }
  })

  if (breadcrumbs.length > 0) breadcrumbs[breadcrumbs.length - 1].last = true

  return {
    breadcrumbs,
    flashMessages: state.app.flashMessages,
    hideActions: state.routing.locationBeforeTransitions.pathname.includes(state.tutorial.route),
  }
}

export default connect(
  mapStateToProps,
  (dispatch) => ({
    markFlashDisplayed: (key) => dispatch({type: 'DISPLAYED_FLASH', param: key}),
    dismissFlash: (key) => dispatch({type: 'DISMISS_FLASH', param: key}),
  })
)(PageTitle)
