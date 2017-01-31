import React from 'react'
import steps from './steps.json'
import TutorialBanner from './TutorialBanner/TutorialBanner'
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
    const showTutorial = this.props.pathname.startsWith(this.props.tutorial.route)
    const tutorialRoute = steps[tutorialStep]['route']
    const tutorialTypes = this.props.types
    const TutorialComponent = components[steps[tutorialStep]['component']]

    return (
      <div>
      {tutorialOpen && !tutorialTypes.includes('TutorialForm') &&
        <TutorialBanner
          resumePath={this.props.tutorial.route}
          showTutorial={showTutorial}
          handleDismiss={this.props.dismissTutorial}
          {...steps[tutorialStep]}/>}
      {tutorialOpen && (tutorialTypes.includes(steps[tutorialStep]['component'])) &&
          <TutorialComponent
            userInput={userInput}
            step={tutorialStep}
            {...steps[tutorialStep]}
            handleNext={() => this.props.showNextStep(tutorialRoute)}
            showTutorial={showTutorial}/>}
      </div>
    )
  }
}

import { actions } from 'features/tutorial'
import { connect } from 'react-redux'

const mapStateToProps = (state) => ({
  tutorial: state.tutorial,
  pathname: state.routing.locationBeforeTransitions.pathname
})

const mapDispatchToProps = ( dispatch ) => ({
  dismissTutorial: () => dispatch(actions.dismissTutorial),
  showNextStep: (route) => dispatch(actions.tutorialNextStep(route))
})

export default connect(
  mapStateToProps,
  mapDispatchToProps
)(Tutorial)
