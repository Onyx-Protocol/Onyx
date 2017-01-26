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
    const tutorialStep = this.props.tutorialStep
    const userInput = this.props.tutorialInputs
    const tutorialOpen = this.props.tutorialOpen
    const tutorialTypes = this.props.types
    const TutorialComponent = components[steps[tutorialStep]['component']]

    return (
      <div>
      {tutorialOpen && (tutorialTypes.includes(steps[tutorialStep]['component'])) &&
          <TutorialComponent
            userInput={userInput}
            step={tutorialStep}
            {...steps[tutorialStep]}
            handleNext={this.props.showNextStep}
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
  tutorialStep: state.tutorial.step,
  tutorialOpen: state.tutorial.isShowing,
  tutorialInputs: state.tutorial.userInputs
})

const mapDispatchToProps = ( dispatch ) => ({
  dismissTutorial: () => dispatch(actions.dismissTutorial),
  showNextStep: () => dispatch(actions.tutorialNextStep)
})

export default connect(
  mapStateToProps,
  mapDispatchToProps
)(Tutorial)
