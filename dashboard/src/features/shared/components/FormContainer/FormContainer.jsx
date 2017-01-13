import React from 'react'
import { ErrorBanner, PageTitle, FormSection, SubmitIndicator } from 'features/shared/components'
import styles from './FormContainer.scss'
import Tutorial from 'features/tutorial/components/Tutorial'

class FormContainer extends React.Component {
  constructor(props) {
    super(props)

    this.showNextStep = this.showNextStep.bind(this)
  }

  showNextStep() {
    if(this.props.tutorialOpen){
      this.props.showNextStep()
    }
  }

  render() {
    return(
      <div className='flex-container'>
        <PageTitle title={this.props.label} />

        <div className={`${styles.main} flex-container`}>
          <div className={styles.content}>
            <form onSubmit={this.props.onSubmit}>
              {this.props.children}

              <FormSection className={styles.submitSection}>
                {this.props.error &&
                  <ErrorBanner
                    title='Error submitting form'
                    error={this.props.error} />}

                <div className={styles.submit}>
                  <button type='submit' className='btn btn-primary' onClick={this.showNextStep} disabled={this.props.submitting || this.props.disabled}>
                    {this.props.submitLabel || 'Submit'}
                  </button>

                  {this.props.showSubmitIndicator && this.props.submitting &&
                    <SubmitIndicator />
                  }
                </div>
              </FormSection>
            </form>
          </div>
          <Tutorial types={['Form']} />
        </div>
      </div>
    )
  }
}

import { actions } from 'features/tutorial'
import { connect } from 'react-redux'

const mapStateToProps = (state) => ({
  tutorialOpen: state.tutorial.isShowing
})

const mapDispatchToProps = ( dispatch ) => ({
  showNextStep: () => dispatch(actions.tutorialNextStep)
})

export default connect(
  mapStateToProps,
  mapDispatchToProps
)(FormContainer)
