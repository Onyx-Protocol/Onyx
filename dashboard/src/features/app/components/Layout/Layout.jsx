import React from 'react'
import styles from './Layout.scss'
import { Link } from 'react-router'
import { connect } from 'react-redux'
import appActions from 'features/app/actions'
import Tutorial from 'features/tutorial/components/Tutorial'
import TutorialHeader from 'features/tutorial/components/TutorialHeader/TutorialHeader'
import { Navigation, SecondaryNavigation, Modal } from '../'

class Layout extends React.Component {

  constructor(props) {
    super(props)

    this.toggleDropdown = this.toggleDropdown.bind(this)
  }

  toggleDropdown(event) {
    event.stopPropagation()
    this.props.toggleDropdown()
  }

  render() {
    let logo = require('images/logo-white.png')

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
                  <img src={require('images/navigation/settings.png')}/>
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
            <TutorialHeader>
              <Tutorial types={['TutorialInfo', 'TutorialModal']}/>
            </TutorialHeader>
          {this.props.children}
        </div>

         <Modal />

         {/* For copyToClipboard(). TODO: move this some place cleaner. */}
         <input
           id='_copyInput'
           onChange={() => 'do nothing'}
           value='dummy'
           style={{display: 'none'}}
         />
      </div>
    )
  }
}

export default connect(
  (state) => ({
    canLogOut: state.authn.authenticationRequired,
    connected: state.authn.connected,
    showDropwdown: state.app.dropdownState == 'open',
  }),
  (dispatch) => ({
    toggleDropdown: () => dispatch(appActions.toggleDropdown),
    closeDropdown: () => dispatch(appActions.closeDropdown),
  })
)(Layout)
