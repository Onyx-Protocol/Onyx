import React from 'react'
import steps from './steps.json'
import TutorialInfo from './TutorialInfo/TutorialInfo'
import TutorialForm from './TutorialForm/TutorialForm'
import TutorialComplete from './TutorialComplete/TutorialComplete'

const components = {
  TutorialInfo,
  TutorialForm,
  TutorialComplete
}

class Tutorial extends React.Component {
  render() {
    const tutorialStep = this.props.tutorial.step
    const userInput = this.props.tutorial.userInputs
    const tutorialOpen = this.props.tutorial.isShowing
    const tutorialRoute = steps[tutorialStep]['route']
    const tutorialTypes = this.props.types
    const TutorialComponent = components[steps[tutorialStep]['component']]

    return (
      <div>
      {tutorialOpen && (tutorialTypes.includes(steps[tutorialStep]['component'])) &&
          <TutorialComponent
            userInput={userInput}
            step={tutorialStep}
            {...steps[tutorialStep]}
            handleNext={() => this.props.showNextStep(tutorialRoute)}
            handleDismiss={this.props.dismissTutorial}
          />
        }
    </div>
    )
  }
}

import { actions } from 'features/tutorial'
import { connect } from 'react-redux'

const mapStateToProps = (state) => ({
  tutorial: state.tutorial
})

const mapDispatchToProps = ( dispatch ) => ({
  dismissTutorial: () => dispatch(actions.dismissTutorial),
  showNextStep: (route) => dispatch(actions.tutorialNextStep(route))
})

export default connect(
  mapStateToProps,
  mapDispatchToProps
)(Tutorial)
