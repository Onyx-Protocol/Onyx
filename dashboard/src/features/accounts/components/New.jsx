import React from 'react'
import { BaseNew, FormContainer } from 'features/shared/components'
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
        .catch((err) => reject({_error: err.message}))
    })
  }

  render() {
    const {
      fields: { alias, tags, root_xpubs, quorum },
      error,
      handleSubmit,
      submitting
    } = this.props

    return(
      <FormContainer
        error={error}
        label='New Account'
        onSubmit={handleSubmit(this.submitWithErrors)}
        submitting={submitting} >

        <TextField title='Alias' placeholder='Alias' fieldProps={alias} />
        <JsonField title='Tags' fieldProps={tags} />
        <KeyConfiguration
          xpubs={root_xpubs}
          quorum={quorum}
          quorumHint='Number of keys required for transfer' />

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

const fields = [ 'alias', 'tags', 'root_xpubs[]', 'quorum' ]

export default BaseNew.connect(
  BaseNew.mapStateToProps('account'),
  BaseNew.mapDispatchToProps('account'),
  reduxForm({
    form: 'newAccountForm',
    fields,
    validate,
    initialValues: {
      tags: '{\n\t\n}',
    }
  })(Form)
)
