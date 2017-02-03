import React from 'react'
import { Link } from 'react-router'
import styles from './TutorialHeader.scss'

class TutorialHeader extends React.Component {

  render() {

    return (
      <div>
        {this.props.tutorial.isShowing &&
          <div className={styles.main}>
            <div className={styles.header}>
              {this.props.currentStep.title}
              {this.props.currentStep.dismiss &&
                <div className={styles.skip}>
                  {!this.props.showTutorial && <Link to={this.props.tutorial.route}>
                    Resume tutorial
                  </Link>}
                  {this.props.showTutorial &&
                  <a onClick={this.props.dismissTutorial}>{this.props.currentStep.dismiss}</a>}
                </div>}
            </div>
            {this.props.showTutorial && this.props.children}
          </div>}
    </div>
    )
  }
}

import { actions } from 'features/tutorial'
import { connect } from 'react-redux'

const mapStateToProps = (state) => ({
  tutorial: state.tutorial,
  currentStep: state.tutorial.currentStep,
  showTutorial: state.routing.locationBeforeTransitions.pathname.startsWith(state.tutorial.route)
})

const mapDispatchToProps = ( dispatch ) => ({
  dismissTutorial: () => dispatch(actions.dismissTutorial)
})

export default connect(
  mapStateToProps,
  mapDispatchToProps
)(TutorialHeader)
