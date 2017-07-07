import React from 'react'
import { Link } from 'react-router'
import styles from './TutorialHeader.scss'

class TutorialHeader extends React.Component {

  render() {
    if(!this.props.tutorial.isShowing || this.props.currentStep.component == 'TutorialModal'){
      return (
        <div>
          {this.props.children}
        </div>
      )
    } else {
      const collapsed = !this.props.showTutorial || this.props.currentStep.component == 'TutorialForm'
      return (
        <div className={styles.container}>
          <div className={`${styles.main} ${collapsed && styles.collapsed}`}>
            <div className={styles.header}>
              {this.props.currentStep.title}
              <div className={styles.skip}>
                {!this.props.showTutorial &&
                  <Link to={this.props.tutorial.route}>
                    Resume tutorial
                  </Link>}

                {this.props.showTutorial &&
                  <a onClick={this.props.dismissTutorial}>
                    {this.props.currentStep.dismiss || 'End tutorial'}
                  </a>}
              </div>
            </div>
            {this.props.showTutorial && this.props.children}
          </div>
        </div>
      )
    }
  }
}

import { connect } from 'react-redux'

const mapStateToProps = (state) => ({
  tutorial: state.tutorial,
  currentStep: state.tutorial.currentStep,
  showTutorial: state.routing.locationBeforeTransitions.pathname.includes(state.tutorial.route)
})

const mapDispatchToProps = ( dispatch ) => ({
  dismissTutorial: () => dispatch({ type: 'DISMISS_TUTORIAL' })
})

export default connect(
  mapStateToProps,
  mapDispatchToProps
)(TutorialHeader)
