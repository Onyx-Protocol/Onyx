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
      fields: { alias, tags, definition, xpubs, quorum },
      error,
      handleSubmit,
      submitting
    } = this.props

    const mockhsmKeys = this.props.mockhsmKeys ? this.props.mockhsmKeys.items : []

    return(
      <div className='form-container'>
        <PageHeader title="New Asset" />

        <form onSubmit={handleSubmit(this.submitWithErrors)}>
          <TextField title='Alias' placeholder='Alias' fieldProps={alias} />
          <TextareaField title="Tags" fieldProps={tags} />
          <TextareaField title="Definition" fieldProps={definition} />
          <KeyConfiguration xpubs={xpubs} quorum={quorum} mockhsmKeys={mockhsmKeys}/>

          {error && <ErrorBanner
            title="There was a problem creating your asset:"
            message={error}/>}

          <button type='submit' className='btn btn-primary' disabled={submitting}>
            Submit
          </button>
        </form>
      </div>
    )
  }
}

const fields = [ 'alias', 'tags', 'definition', 'xpubs[]', 'quorum' ]
export default reduxForm({
  form: 'newAssetForm',
  fields,
  initialValues: {
    tags: '{}',
    definition: '{}',
  }
})(Form)
