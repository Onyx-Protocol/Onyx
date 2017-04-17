import React from 'react'
import { BaseNew, FormContainer, FormSection, TextField, CheckboxField } from 'features/shared/components'
import { policyOptions } from 'features/accessControl/constants'
import { reduxForm } from 'redux-form'
import { actions } from 'features/accessControl'

class NewToken extends React.Component {
  render() {
    const {
      fields: { guardData, policies },
      error,
      handleSubmit,
      submitting
    } = this.props

    return(
      <FormContainer
        error={error}
        label='New access token'
        onSubmit={handleSubmit(this.props.submitForm)}
        submitting={submitting} >

        <FormSection title='Token information'>
          <TextField title='Token Name' fieldProps={guardData.id} autoFocus={true} />
        </FormSection>
        <FormSection title='Policy'>
          {policyOptions.map(option => {
            return <CheckboxField key={option.label}
              title={option.label}
              hint={option.hint}
              fieldProps={policies[option.value]} />
          })}
        </FormSection>

      </FormContainer>
    )
  }
}

const fields = [
  'guardType',
  'guardData.id',
  'policies.client-readwrite',
  'policies.client-readonly',
  'policies.network',
  'policies.monitoring',
]

const validate = values => {
  const errors = {}

  // if (!values.policy) {
  //   errors.policy = 'Policy is required'
  // }
  if (!values.guardData.id) {
    errors.guardData = {id: 'Token name is required'}
  }

  return errors
}

const mapDispatchToProps = (dispatch) => ({
  submitForm: (data) => dispatch(actions.submitTokenForm(data))
})

export default BaseNew.connect(
  BaseNew.mapStateToProps('accessControl'),
  mapDispatchToProps,
  reduxForm({
    form: 'newAccessGrantForm',
    fields,
    validate
  })(NewToken)
)
