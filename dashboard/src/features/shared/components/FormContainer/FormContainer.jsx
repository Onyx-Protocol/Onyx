import React from 'react'
import PageHeader from 'components/PageHeader/PageHeader'
import { ErrorBanner } from 'features/shared/components'
import styles from './FormContainer.scss'

class FormContainer extends React.Component {
  render() {
    return(
      <div className={styles.main}>
        <PageHeader title={this.props.label} />

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
    )
  }
}

export default FormContainer
