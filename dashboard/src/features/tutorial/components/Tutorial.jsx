import React from 'react'
import steps from './steps.json'
import Description from './Description/Description'
import Success from './Success/Success'
import Form from './Form/Form'

const components = {
  Description,
  Success,
  Form
}

class Tutorial extends React.Component {

  render() {
    const tutorialStep = this.props.tutorialStep
    const tutorialOpen = this.props.tutorialOpen
    const tutorialTypes = this.props.types
    const TutorialComponent = components[steps[tutorialStep]['component']]

    return (
      <span>
      {tutorialOpen && (tutorialTypes.includes(steps[tutorialStep]['component'])) &&
          <TutorialComponent
            step={tutorialStep}
            button={steps[tutorialStep]['button']}
            title={steps[tutorialStep]['title']}
            content={steps[tutorialStep]['content']}
            dismiss={steps[tutorialStep]['dismiss']}
            route={steps[tutorialStep]['route']}
            handleNext={this.props.showNextStep}
            handleDismiss={this.props.dismissTutorial}
          />
        }
    </span>
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
  dismissTutorial: () => dispatch(actions.dismissTutorial),
  showNextStep: () => dispatch(actions.tutorialNextStep)
})

export default connect(
  mapStateToProps,
  mapDispatchToProps
)(Tutorial)
