import React from 'react'
import { BaseNew, FormContainer, FormSection, CheckboxField } from 'features/shared/components'
import { policyOptions } from 'features/accessControl/constants'
import { reduxForm } from 'redux-form'
import actions from 'features/accessControl/actions'
import { isAccessToken } from 'features/accessControl/selectors'

class EditPolicies extends React.Component {
  render() {
    const item = this.props.item
    const {
      fields: { policies },
      error,
      handleSubmit,
      submitting
    } = this.props

    const label = <span>Edit {isAccessToken(item) ? <code>{item.name}</code> : 'certificate'}</span>

    return(
      <FormContainer
        error={error}
        label={label}
        onSubmit={handleSubmit(this.props.submitForm)}
        submitting={submitting} >

        {!isAccessToken(item) && <FormSection title='Certificate Info'>
          <pre>
            {JSON.stringify(item.guardData, '  ', 2)}
          </pre>
        </FormSection>}

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

const mapStateToProps = (state, ownProps) => ({
  item: state.accessControl.items[ownProps.params.id]
})

const mapDispatchToProps = (dispatch) => ({
  submitForm: (data) => dispatch(actions.editPolicies(data))
})

const initialValues = (state, ownProps) => {
  const item = ownProps.item
  if (!item) { return {} }

  const policies = item.policies
  const fields = {
    initialValues: {
      grant: item,
      policies: {
        ['client-readwrite']: policies.indexOf('client-readwrite') >= 0,
        ['client-readonly']: policies.indexOf('client-readonly') >= 0,
        network: policies.indexOf('network') >= 0,
        monitoring: policies.indexOf('monitoring') >= 0,
      }
    }
  }

  return fields
}

export default BaseNew.connect(
  mapStateToProps,
  mapDispatchToProps,
  reduxForm({
    form: 'editPoliciesForm',
    fields: [
      'grant',
      'policies.client-readwrite',
      'policies.client-readonly',
      'policies.network',
      'policies.monitoring',
    ],
  }, initialValues)(EditPolicies)
)
