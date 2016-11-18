import React from 'react'
import { BaseNew, FormContainer, FormSection } from 'features/shared/components'
import { TextField } from 'components/Common'
import { reduxForm } from 'redux-form'

class New extends React.Component {
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
      fields: { alias },
      error,
      handleSubmit,
      submitting
    } = this.props

    return(
      <FormContainer
        error={error}
        label='New MockHSM key'
        onSubmit={handleSubmit(this.submitWithErrors)}
        submitting={submitting} >

        <FormSection title='Key Information'>
          <TextField title='Alias' placeholder='Alias' fieldProps={alias} autoFocus={true} />
        </FormSection>
      </FormContainer>
    )
  }
}

const fields = [ 'alias' ]
export default BaseNew.connect(
  BaseNew.mapStateToProps('mockhsm'),
  BaseNew.mapDispatchToProps('mockhsm'),
  reduxForm({
    form: 'newMockHsmKey',
    fields,
  })(New)
)
