import React from 'react'
import { BaseNew, FormContainer, FormSection, CheckboxField } from 'features/shared/components'
import { policyOptions } from 'features/accessControl/constants'
import { reduxForm } from 'redux-form'
import actions from 'features/accessControl/actions'

class EditPolicies extends React.Component {
  render() {
    const {
      fields: { policies },
      error,
      handleSubmit,
      submitting
    } = this.props

    return(
      <FormContainer
        error={error}
        label='Edit policies'
        onSubmit={handleSubmit(this.props.submitForm)}
        submitting={submitting} >

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

const mapDispatchToProps = (dispatch) => ({
  submitForm: (data) => { /* todo */ }
})

const initialValues = (state, ownProps) => {
  const item = state.accessControl.items[ownProps.params.id]
  if (!item) { return {} }

  const policies = item.policies
  const fields = {
    initialValues: {
      policies: {
        ['client-readwrite']: policies.indexOf('client-readwrite') >= 0,
        ['client-readonly']: policies.indexOf('client-readonly') >= 0,
        network: policies.indexOf('network') >= 0,
        monitoring: policies.indexOf('monitoring') >= 0,
      }
    }
  }
  console.log(fields);

  return fields
}

export default BaseNew.connect(
  BaseNew.mapStateToProps('accessControl'),
  mapDispatchToProps,
  reduxForm({
    form: 'editPoliciesForm',
    fields: [
      'policies.client-readwrite',
      'policies.client-readonly',
      'policies.network',
      'policies.monitoring',
    ],
  }, initialValues)(EditPolicies)
)
