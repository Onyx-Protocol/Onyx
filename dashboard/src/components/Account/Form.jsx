import React from 'react'
import PageHeader from "../PageHeader/PageHeader"
import {
  TextField,
  TextareaField,
  KeyConfiguration,
  ErrorBanner
} from "../Common"
import { reduxForm } from 'redux-form'

class Form extends React.Component {
  constructor(props) {
    super(props)

    this.submitWithErrors = this.submitWithErrors.bind(this)
  }

  submitWithErrors(data) {
    return new Promise((resolve, reject) => {
      this.props.submitForm(data)
        .catch((err) => reject({_error: err.message}))
    })
  }

  render() {
    const {
      fields: { alias, tags, xpubs, quorum },
      error,
      handleSubmit,
      submitting
    } = this.props

    const mockhsmKeys = this.props.mockhsmKeys ? this.props.mockhsmKeys.items : []

    return(
      <div className='form-container'>
        <PageHeader title="New Account" />

        <form onSubmit={handleSubmit(this.submitWithErrors)}>
          <TextField title='Alias' placeholder='Alias' fieldProps={alias} />
          <TextareaField title="Tags" fieldProps={tags} />
          <KeyConfiguration xpubs={xpubs} quorum={quorum} mockhsmKeys={mockhsmKeys}/>

          {error && <ErrorBanner
            title="There was a problem creating your account:"
            message={error}/>}

          <button type='submit' className='btn btn-primary' disabled={submitting}>
            Submit
          </button>
        </form>
      </div>
    )
  }
}

const fields = [ 'alias', 'tags', 'xpubs[]', 'quorum' ]
export default reduxForm({
  form: 'newAccountForm',
  fields,
  initialValues: {
    tags: '{}',
  }
})(Form)
