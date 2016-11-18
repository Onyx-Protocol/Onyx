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
      fields: { alias, tags, xpubs, quorum },
      error,
      handleSubmit,
      submitting
    } = this.props

    return(
      <FormContainer
        error={error}
        label='New account'
        onSubmit={handleSubmit(this.submitWithErrors)}
        submitting={submitting} >

        <FormSection title='Account Information'>
          <TextField title='Alias' placeholder='Alias' fieldProps={alias} autoFocus={true} />
          <JsonField title='Tags' fieldProps={tags} />
        </FormSection>

        <FormSection title='Keys and Signing'>
          <KeyConfiguration
            xpubs={xpubs}
            quorum={quorum}
            quorumHint='Number of keys required for transfer' />
        </FormSection>
      </FormContainer>
    )
  }
}

const validate = values => {
  const errors = {}

  const tagError = JsonField.validator(values.tags)
  if (tagError) { errors.tags = tagError }

  return errors
}

const fields = [
  'alias',
  'tags',
  'xpubs[].value',
  'xpubs[].type',
  'quorum'
]

export default BaseNew.connect(
  BaseNew.mapStateToProps('account'),
  BaseNew.mapDispatchToProps('account'),
  reduxForm({
    form: 'newAccountForm',
    fields,
    validate,
    initialValues: {
      tags: '{\n\t\n}',
      quorum: 1,
    }
  })(Form)
)
