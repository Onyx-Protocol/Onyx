import { BaseNew, FormContainer, FormSection, HiddenField } from 'features/shared/components'
import { TextField } from 'components/Common'
import React from 'react'
import { reduxForm } from 'redux-form'
import { humanize } from 'utility/string'

const clientType = 'client_access_token'
const networkType = 'network_access_token'

const Form = class Form extends React.Component {
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
  BaseNew.mapStateToProps(clientType),
  BaseNew.mapDispatchToProps(clientType),
  reduxForm({
    form: 'newTokenForm',
    fields,
    initialValues: {
      type: 'client',
    }
  })(Form)
)

export const NewNetworkToken = BaseNew.connect(
  BaseNew.mapStateToProps(networkType),
  BaseNew.mapDispatchToProps(networkType),
  reduxForm({
    form: 'newTokenForm',
    fields,
    initialValues: {
      type: 'network',
    }
  })(Form)
)
