import React from 'react'
import styles from './Tutorial.scss'
import steps from './steps.json'

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
    this.setState({step: this.state.step + 1})
  }

  dismissTutorial() {
    this.setState({open: false})
  }

  render() {
    const nextButton = <div className={styles.next}>
      <button key='showNext' className='btn btn-primary' onClick={this.showNextStep}>
        {steps[this.state.step]['button']}
      </button>
    </div>
    let open = this.state.open

    return (
      <div>
      {open &&
          <div className={styles.container}>
            <div className={styles.header}>
              {steps[this.state.step]['title']}
              <div className={styles.skip}>
                <a onClick={this.dismissTutorial}>{steps[this.state.step]['dismiss']}</a>
              </div>
            </div>
            <div className={styles.content}>
              {steps[this.state.step]['content']}

              {nextButton && nextButton}
            </div>
          </div>
        }
    </div>
    )
  }
}

export default Tutorial
