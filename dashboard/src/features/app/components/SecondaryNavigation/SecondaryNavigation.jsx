import React from 'react'
import { Link } from 'react-router'
import { connect } from 'react-redux'
import { navIcon } from '../../utils'
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
              {navIcon('core', styles)}
              Core status
            </Link>
          </li>
          <li>
            <Link to='/access-control' activeClassName={styles.active}>
              {navIcon('network', styles)}
              Access Control
            </Link>
          </li>

          {this.props.canLogOut && <li className={styles.logOut}>
            <a href='#' onClick={this.logOut}>
              {navIcon('logout', styles)}
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
    canLogOut: state.authn.authenticationRequired,
  }),
  (dispatch) => ({
    logOut: () => dispatch({type: 'USER_LOG_OUT'})
  })
)(SecondaryNavigation)
