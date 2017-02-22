import React from 'react'
import TutorialInfo from './TutorialInfo/TutorialInfo'
import TutorialForm from './TutorialForm/TutorialForm'
import TutorialModal from './TutorialModal/TutorialModal'

const components = {
  TutorialInfo,
  TutorialForm,
  TutorialModal
}

class Tutorial extends React.Component {
  render() {
    const userInput = this.props.tutorial.userInputs
    const tutorialOpen = this.props.tutorial.isShowing
    const tutorialRoute = this.props.currentStep['route']
    const tutorialTypes = this.props.types
    const TutorialComponent = components[this.props.currentStep['component']]

    return (
      <div>
      {tutorialOpen && (tutorialTypes.includes(this.props.currentStep['component'])) &&
        <TutorialComponent
          userInput={userInput}
          {...this.props.currentStep}
          handleNext={() => this.props.showNextStep(tutorialRoute)}/>}
      </div>
    )
  }
}

import { actions } from 'features/tutorial'
import { connect } from 'react-redux'

const mapStateToProps = (state) => ({
  currentStep: state.tutorial.currentStep,
  tutorial: state.tutorial
})

const mapDispatchToProps = ( dispatch ) => ({
  showNextStep: (route) => dispatch(actions.tutorialNextStep(route))
})

export default connect(
  mapStateToProps,
  mapDispatchToProps
)(Tutorial)
