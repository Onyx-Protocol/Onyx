import React from 'react'
import styles from './TutorialComplete.scss'


class TutorialComplete extends React.Component {

  render() {
    const userInput = this.props.userInput

    return (
      <div className={styles.main}>
        <div className={styles.backdrop}></div>
          <div className={styles.content}>
            <div className={styles.header}>
              5-minute Tutorial completed!
            </div>
            <div className={styles.text}>
              <p>
                In this tutorial, you learned how to:<br />
              </p>
              <p>
                1. create and issue {userInput['asset']['alias']} assets<br />
                2. create accounts for {userInput['accounts'][0]['alias']} and {userInput['accounts'][1]['alias']}<br />
                3. build, sign and submit transactions<br />
              </p>
              <p>
                  If you need to revisit this tutorial, you can click Tutorial in
                  the sidenav. For detailed information on how Chain
                  Core works, please take a look at the <a href='/docs' target='_blank'>
                    Developer Documentation
                  </a>.
              </p>
            </div>
            <button onClick={this.props.dismissTutorial} className={`btn btn-primary ${styles.tutorialButton}`}>{this.props.dismiss}</button>
          </div>
      </div>
    )
  }
}

import { actions } from 'features/tutorial'
import { connect } from 'react-redux'

const mapStateToProps = (state) => ({
  tutorialRoute: state.tutorial.route,
  currentStep: state.tutorial.currentStep,
  showTutorial: state.routing.locationBeforeTransitions.pathname.startsWith(state.tutorial.route)
})

const mapDispatchToProps = ( dispatch ) => ({
  dismissTutorial: () => dispatch(actions.dismissTutorial)
})

export default connect(
  mapStateToProps,
  mapDispatchToProps
)(TutorialComplete)
