import React from 'react'
import steps from './steps.json'
import Description from './Description/Description'
import Success from './Success/Success'

const components = {
  Description,
  Success
}

class Tutorial extends React.Component {
  constructor(props) {
    super(props)

    // TODO: examine renaming and refactoring for clarity. Consider moving
    // away from local state if possible.
    this.state = {
      open: true,
      step: 0,
    }

    this.showNextStep = this.showNextStep.bind(this)
    this.dismissTutorial = this.dismissTutorial.bind(this)
  }

  showNextStep() {
    if(this.state.step == steps.length - 1){
      this.setState({open: false})
    } else {
      this.setState({step: this.state.step + 1})
    }
  }

  dismissTutorial() {
    this.setState({open: false})
  }

  render() {
    const TutorialComponent = components[steps[this.state.step]['component']]
    return (
      <div>
      {this.state.open &&
          <TutorialComponent
            step={this.state.step}
            button={steps[this.state.step]['button']}
            title={steps[this.state.step]['title']}
            content={steps[this.state.step]['content']}
            dismiss={steps[this.state.step]['dismiss']}
            handleNext={this.showNextStep}
            handleDismiss={this.dismissTutorial}
          />
        }
    </div>
    )
  }
}

export default Tutorial
