import React from 'react'
import steps from './steps.json'
import Description from './Description/Description'
import Success from './Success/Success'

const components = {
  Description,
  Success
}

class Tutorial extends React.Component {

  render() {
    const tutorialStep = this.props.tutorialStep
    const tutorialOpen = this.props.tutorialOpen
    const TutorialComponent = components[steps[tutorialStep]['component']]

    return (
      <div>
      {tutorialOpen &&
          <TutorialComponent
            step={tutorialStep}
            button={steps[tutorialStep]['button']}
            title={steps[tutorialStep]['title']}
            content={steps[tutorialStep]['content']}
            dismiss={steps[tutorialStep]['dismiss']}
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
  tutorialOpen: state.tutorial.isShowing
})

const mapDispatchToProps = ( dispatch ) => ({
  dismissTutorial: () => dispatch(actions.toggleTutorial),
  showNextStep: () => dispatch(actions.tutorialNextStep)
})

export default connect(
  mapStateToProps,
  mapDispatchToProps
)(Tutorial)
