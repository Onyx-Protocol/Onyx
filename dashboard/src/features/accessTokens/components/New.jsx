import { BaseNew, FormContainer, FormSection, HiddenField, TextField } from 'features/shared/components'
import React from 'react'
import { reduxForm } from 'redux-form'
import { humanize } from 'utility/string'

const Form = class Form extends React.Component {
  constructor(props) {
    super(props)

    this.submitWithErrors = this.submitWithErrors.bind(this)
  }

  submitWithErrors(data) {
    return this.props.submitForm(data)
      .catch((err) => { throw {_error: err} })
  }

  render() {
    const {
      fields: { id, type },
      error,
      handleSubmit,
      submitting
    } = this.props

    const label = humanize(this.props.type)

    return(<FormContainer
      error={error}
      label={`New ${label}`}
      onSubmit={handleSubmit(this.submitWithErrors)}
      submitting={submitting} >

      <FormSection title='Token Information'>
        <TextField
          title='Token ID'
          placeholder='Token ID'
          fieldProps={id}
          autoFocus={true}
          hint='Valid characters include letters, numbers, underscores, and hyphens.'
        />
        <HiddenField fieldProps={type} />
      </FormSection>
    </FormContainer>)
  }
}

const fields = [ 'id', 'type' ]

export const NewClientToken = BaseNew.connect(
  BaseNew.mapStateToProps('clientAccessToken'),
  BaseNew.mapDispatchToProps('clientAccessToken'),
  reduxForm({
    form: 'newTokenForm',
    fields,
    initialValues: {
      type: 'client',
    }
  })(Form)
)

export const NewNetworkToken = BaseNew.connect(
  BaseNew.mapStateToProps('networkAccessToken'),
  BaseNew.mapDispatchToProps('networkAccessToken'),
  reduxForm({
    form: 'newTokenForm',
    fields,
    initialValues: {
      type: 'network',
    }
  })(Form)
)
