import React from 'react'
import { BaseNew, CheckboxField } from 'features/shared/components'
import { policyOptions } from 'features/accessControl/constants'
import { reduxForm } from 'redux-form'
import actions from 'features/accessControl/actions'
import styles from './EditPolicies.scss'

class EditPolicies extends React.Component {
  render() {
    const {
      fields: { policies },
      handleSubmit,
    } = this.props

    return(
      <div className={styles.main}>
        {policyOptions.map(option => {
          return <CheckboxField key={option.label}
            title={option.label}
            hint={option.hint}
            fieldProps={policies[option.value]} />
        })}

        <button className='btn btn-primary' onClick={handleSubmit(this.props.submitForm)}>Save</button>
      </div>
    )
  }
}

const mapDispatchToProps = (dispatch) => ({
  submitForm: (data) => dispatch(actions.editPolicies(data))
})

const initialValues = (state, ownProps) => {
  const item = ownProps.item
  if (!item) { return {} }

  const fields = {
    initialValues: {
      grant: item,
      policies: policyOptions.reduce((memo, p) => {
        memo[p.value] = item.policies.indexOf(p.value) >= 0
        return memo
      }, {}),
    }
  }

  return fields
}

export default BaseNew.connect(
  () => ({}),
  mapDispatchToProps,
  reduxForm({
    form: 'editPoliciesForm',
    fields: ['grant'].concat(policyOptions.map(p => `policies.${p.value}`)),
  }, initialValues)(EditPolicies)
)
