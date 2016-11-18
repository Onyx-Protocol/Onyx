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
      fields: { alias, filter },
      error,
      handleSubmit,
      submitting
    } = this.props

    return(
      <FormContainer
        error={error}
        label='New transaction feed'
        onSubmit={handleSubmit(this.submitWithErrors)}
        submitting={submitting} >

        <FormSection title='Feed Information'>
          <TextField title='Alias' placeholder='Alias' fieldProps={alias} autoFocus={true} />
          <TextField title='Filter' placeholder='Filter' fieldProps={filter} />
        </FormSection>
      </FormContainer>
    )
  }
}

const fields = [ 'alias', 'filter' ]
export default BaseNew.connect(
  BaseNew.mapStateToProps('transactionFeed'),
  BaseNew.mapDispatchToProps('transactionFeed'),
  reduxForm({
    form: 'newTxFeed',
    fields,
  })(New)
)
