import React from 'react'
import { BaseNew, FormContainer, FormSection } from 'features/shared/components'
import {
  TextField,
  JsonField,
  KeyConfiguration,
} from 'components/Common'
import { reduxForm } from 'redux-form'

class Form extends React.Component {
  constructor(props) {
    super(props)

    this.submitWithErrors = this.submitWithErrors.bind(this)
  }

  submitWithErrors(data) {
    return new Promise((resolve, reject) => {
      this.props.submitForm(data)
        .catch((err) => reject({_error: err}))
    })
  }

  render() {
    const {
      fields: { alias, tags, definition, xpubs, quorum },
      error,
      handleSubmit,
      submitting
    } = this.props

    return(
      <FormContainer
        error={error}
        label='New asset'
        onSubmit={handleSubmit(this.submitWithErrors)}
        submitting={submitting} >

        <FormSection title='Asset Information'>
          <TextField title='Alias' placeholder='Alias' fieldProps={alias} autoFocus={true} />
          <JsonField title='Tags' fieldProps={tags} />
          <JsonField title='Definition' fieldProps={definition} />
        </FormSection>

        <FormSection title='Keys and Signing'>
          <KeyConfiguration
            xpubs={xpubs}
            quorum={quorum}
            quorumHint='Number of signatures required to issue' />
        </FormSection>

      </FormContainer>
    )
  }
}

const validate = values => {
  const errors = {}

  const jsonFields = ['tags', 'definition']
  jsonFields.forEach(key => {
    const fieldError = JsonField.validator(values[key])
    if (fieldError) { errors[key] = fieldError }
  })

  return errors
}

const fields = [
  'alias',
  'tags',
  'definition',
  'xpubs[].value',
  'xpubs[].type',
  'quorum'
]
export default BaseNew.connect(
  BaseNew.mapStateToProps('asset'),
  BaseNew.mapDispatchToProps('asset'),
  reduxForm({
    form: 'newAssetForm',
    fields,
    validate,
    initialValues: {
      tags: '{\n\t\n}',
      definition: '{\n\t\n}',
      quorum: 1,
    }
  })(Form)
)
