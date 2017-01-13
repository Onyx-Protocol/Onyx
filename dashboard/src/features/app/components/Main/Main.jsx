import React from 'react'
import styles from './Main.scss'
import { Link } from 'react-router'
import { connect } from 'react-redux'
import actions from 'actions'
import Tutorial from 'features/tutorial/components/Tutorial'
import { Navigation, SecondaryNavigation } from '../'

class Main extends React.Component {

  constructor(props) {
    super(props)

    this.toggleDropdown = this.toggleDropdown.bind(this)
  }

  toggleDropdown(event) {
    event.stopPropagation()
    this.props.toggleDropdown()
  }

  render() {
    let logo = require('assets/images/logo-white.png')

    return (
      <div className={styles.main}
           onClick={this.props.closeDropdown} >
        <div className={styles.sidebar}>
          <div className={styles.sidebarContent}>
            <div className={styles.logo}>
              <Link to={'/'}>
                <img src={logo} className={styles.brand_image} />
              </Link>

              <span>
                <span className={styles.settings} onClick={this.toggleDropdown}>
                  <img src={require('assets/images/navigation/settings.png')}/>
                </span>
                {this.props.showDropwdown && <SecondaryNavigation />}
              </span>
            </div>

            <Navigation />
          </div>
        </div>

        <div className={`${styles.content} flex-container`}>
          {!this.props.connected && <div className={styles.connectionIssue}>
            There was an issue connecting to Chain Core. Please check your connection while dashboard attempts to reconnect.
          </div>}

          <Tutorial types={['Description', 'Success', 'Complete']} />

          {this.props.children}
        </div>
      </div>
    )
  }
}

export default connect(
  (state) => ({
    canLogOut: state.core.requireClientToken,
    connected: state.core.connected,
    showDropwdown: state.app.dropdownState == 'open',
  }),
  (dispatch) => ({
    toggleDropdown: () => dispatch(actions.app.toggleDropdown),
    closeDropdown: () => dispatch(actions.app.closeDropdown),
  })
)(Main)
