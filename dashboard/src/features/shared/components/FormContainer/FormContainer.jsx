import React from 'react'
import { ErrorBanner, PageTitle } from 'features/shared/components'
import styles from './FormContainer.scss'

class FormContainer extends React.Component {
  render() {
    return(
      <div>
        <PageTitle title={this.props.label} />

        <div className={styles.main}>
          <form onSubmit={this.props.onSubmit}>
            {this.props.children}

            {this.props.error &&
              <ErrorBanner
                title='Error creating key'
                message={this.props.error.toString()} />}

            <button type='submit' className='btn btn-primary' disabled={this.props.submitting}>
              Submit
            </button>
          </form>
        </div>
      </div>
    )
  }
}

export default FormContainer
