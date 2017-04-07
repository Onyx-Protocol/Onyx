import React from 'react'
import { BaseNew, FormContainer, FormSection, TextField, SelectField } from 'features/shared/components'
import { policyOptions } from 'features/accessControl/constants'
import { reduxForm } from 'redux-form'
import { actions } from 'features/accessControl'
import styles from './NewCertificate.scss'

class NewCertificate extends React.Component {
  constructor(props) {
    super(props)
    this.props.fields.subject.addField()
  }

  render() {
    const {
      fields: { subject, policy },
      error,
      handleSubmit,
      submitting
    } = this.props

    return(
      <FormContainer
        error={error}
        label='Add certificate grant'
        onSubmit={handleSubmit(this.props.submitForm)}
        submitting={submitting} >

        <FormSection title='Certificate subject'>
          {subject.map((line, index) =>
            <div key={index} className={styles.subjectField}>
              <TextField title='Field Name' fieldProps={line.key} autoFocus={true} />
              <TextField title='Field Value' fieldProps={line.value} />
              <button
                className='btn btn-danger btn-xs'
                tabIndex='-1'
                type='button'
                onClick={() => subject.removeField(index)}
              >
                Remove
              </button>
            </div>
          )}
          <button
            type='button'
            className='btn btn-default'
            onClick={() => subject.addField()}
          >
            Add Field
          </button>
        </FormSection>
        <FormSection title='Policy'>
          <SelectField options={policyOptions}
            title='Policy'
            hint='Available policies are:
* `client-readwrite`: full access to the Client API
* `client-readonly`: access to read-only Client endpoints
* `network`: access to the Network API
* `monitoring`: access to monitoring-specific endpoints'
            fieldProps={policy} />
        </FormSection>

      </FormContainer>
    )
  }
}

const fields = [
  'guardType',
  'subject[].key',
  'subject[].value',
  'policy',
]

const validate = values => {
  const errors = {}

  if (!values.policy) {
    errors.policy = 'Policy is required'
  }

  return errors
}

const mapDispatchToProps = (dispatch) => ({
  submitForm: (data) => dispatch(actions.submitCertificateForm(data))
})

export default BaseNew.connect(
  BaseNew.mapStateToProps('accessControl'),
  mapDispatchToProps,
  reduxForm({
    form: 'newAccessGrantForm',
    fields,
    validate
  })(NewCertificate)
)
