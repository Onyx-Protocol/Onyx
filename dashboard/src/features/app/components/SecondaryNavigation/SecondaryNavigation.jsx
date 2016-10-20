import React from 'react'
import { Link } from 'react-router'
import { connect } from 'react-redux'
import { NavigationItem as CoreNavigationItem } from 'features/core/components'
import actions from 'actions'
import styles from './SecondaryNavigation.scss'

class SecondaryNavigation extends React.Component {
  constructor(props) {
    super(props)

    this.logOut = this.logOut.bind(this)
  }

  logOut(event) {
    event.preventDefault()
    this.props.logOut()
  }

  render() {
    return (
      <div className={styles.main}>
        <ul className={styles.navigation}>
          <li className={styles.navigationTitle}>settings</li>

          <li>
            <Link to='/core' activeClassName={styles.active}>
              <CoreNavigationItem externalStyles={styles.glyphicon}/>
            </Link>
          </li>
          <li>
            <Link to='/access_tokens/client' activeClassName={styles.active}>
              <span className={`glyphicon glyphicon-user ${styles.glyphicon}`} />
              Client Tokens
            </Link>
          </li>
          <li>
            <Link to='/access_tokens/network' activeClassName={styles.active}>
              <span className={`glyphicon glyphicon-globe ${styles.glyphicon}`} />
              Network Tokens
            </Link>
          </li>

          {this.props.canLogOut && <li className={styles.logOut}>
            <a href='#' onClick={this.logOut}>
              <span className={`glyphicon glyphicon-log-out ${styles.glyphicon}`} />
              Log Out
            </a>
          </li>}
        </ul>
      </div>
    )
  }
}

export default connect(
  (state) => ({
    canLogOut: state.core.requireClientToken,
  }),
  (dispatch) => ({
    logOut: () => dispatch(actions.core.clearSession())
  })
)(SecondaryNavigation)
