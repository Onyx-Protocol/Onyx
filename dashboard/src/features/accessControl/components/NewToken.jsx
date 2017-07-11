import React from 'react'
import { BaseNew, FormContainer, FormSection, TextField, CheckboxField } from 'features/shared/components'
import { policyOptions } from 'features/accessControl/constants'
import { reduxForm } from 'redux-form'
import actions from 'features/accessControl/actions'

class NewToken extends React.Component {
  isNotValid(fields) {
    if (fields.guardData.id.value.trim() == ''){
      return true
    }
    return false
  }
  render() {
    const {
      fields: { guardData, policies },
      error,
      handleSubmit,
      submitting
    } = this.props

    return(
      <FormContainer
        disabled={this.isNotValid(this.props.fields)}
        error={error}
        label='New access token'
        onSubmit={handleSubmit(this.props.submitForm)}
        submitting={submitting} >

        <FormSection title='Token information'>
          <TextField title='Token Name' fieldProps={guardData.id} autoFocus={true} />
        </FormSection>
        <FormSection title='Policy'>
          {policyOptions.map(option => {
            if (option.hidden) return

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

const mapDispatchToProps = (dispatch) => ({
  submitForm: (data) => dispatch(actions.submitTokenForm(data))
})

export default BaseNew.connect(
  BaseNew.mapStateToProps('accessControl'),
  mapDispatchToProps,
  reduxForm({
    form: 'newAccessGrantForm',
    fields: [
      'guardType',
      'guardData.id',
    ].concat(
      policyOptions.map(p => `policies.${p.value}`)
    )
  })(NewToken)
)
