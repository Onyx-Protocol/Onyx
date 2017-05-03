import React from 'react'
import { BaseNew, CheckboxField } from 'features/shared/components'
import { policyOptions } from 'features/accessControl/constants'
import { hasProtectedGrant } from 'features/accessControl/selectors'
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
          if (option.hidden) return

          const isProtected = hasProtectedGrant(this.props.item.grants, option.value)
          return <CheckboxField key={option.label}
            title={option.label}
            hint={option.hint}
            fieldProps={{
              ...policies[option.value],
              disabled: isProtected,
            }} />
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
        const policyIndex = item.grants.findIndex(grant => grant.policy == p.value)
        memo[p.value] = policyIndex >= 0
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
