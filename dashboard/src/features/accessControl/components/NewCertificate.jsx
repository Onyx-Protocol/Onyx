import React from 'react'
import { BaseNew, FormContainer, FormSection, SelectField, TextField, CheckboxField } from 'features/shared/components'
import { policyOptions, subjectFieldOptions } from 'features/accessControl/constants'
import { reduxForm } from 'redux-form'
import actions from 'features/accessControl/actions'
import styles from './NewCertificate.scss'

class NewCertificate extends React.Component {
  constructor(props) {
    super(props)
    this.props.fields.subject.addField()
  }

  render() {
    const {
      fields: { subject, policies },
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
            <div key={index} className={`NewCertificateSubjectField ${styles.subjectField}`}>
              <SelectField title='Field Name' options={subjectFieldOptions} fieldProps={line.key} />
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
  submitForm: (data) => dispatch(actions.submitCertificateForm(data))
})

export default BaseNew.connect(
  BaseNew.mapStateToProps('accessControl'),
  mapDispatchToProps,
  reduxForm({
    form: 'newAccessGrantForm',
    fields: [
      'guardType',
      'subject[].key',
      'subject[].value'
    ].concat(
      policyOptions.map(p => `policies.${p.value}`)
    ),
  })(NewCertificate)
)
