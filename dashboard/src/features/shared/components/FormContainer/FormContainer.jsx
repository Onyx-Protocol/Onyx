import React from 'react'
import { ErrorBanner, PageTitle, FormSection } from 'features/shared/components'
import styles from './FormContainer.scss'

class FormContainer extends React.Component {
  render() {
    return(
      <div>
        <PageTitle title={this.props.label} />

        <div className={`${styles.main}`}>
          <form onSubmit={this.props.onSubmit}>
            {this.props.children}

            <FormSection className={styles.submitSection}>
              {this.props.error &&
                <ErrorBanner
                  title='Error creating key'
                  message={this.props.error.toString()} />}

              <button type='submit' className={`btn btn-primary ${styles.submit}`} disabled={this.props.submitting}>
                {this.props.submitLabel || 'Submit'}
              </button>
            </FormSection>
          </form>
        </div>
      </div>
    )
  }
}

export default FormContainer
