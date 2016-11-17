import React from 'react'
import { ErrorBanner, PageTitle, FormSection, SubmitIndicator } from 'features/shared/components'
import styles from './FormContainer.scss'

class FormContainer extends React.Component {
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
                  <button type='submit' className='btn btn-primary' disabled={this.props.submitting || this.props.disabled}>
                    {this.props.submitLabel || 'Submit'}
                  </button>

                  {this.props.showSubmitIndicator && this.props.submitting &&
                    <SubmitIndicator />
                  }
                </div>
              </FormSection>
            </form>
          </div>
        </div>
      </div>
    )
  }
}

export default FormContainer
