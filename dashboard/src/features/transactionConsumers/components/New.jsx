import React from 'react'
import { BaseNew, FormContainer } from 'features/shared/components'
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
        .catch((err) => reject({_error: err.message}))
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
        label='New Transaction Consumer'
        onSubmit={handleSubmit(this.submitWithErrors)}
        submitting={submitting} >

          <TextField title='Alias' placeholder='Alias' fieldProps={alias} />
          <TextField title='Filter' placeholder='Filter' fieldProps={filter} />

      </FormContainer>
    )
  }
}

const fields = [ 'alias', 'filter' ]
export default BaseNew.connect(
  BaseNew.mapStateToProps('transactionConsumer'),
  BaseNew.mapDispatchToProps('transactionConsumer'),
  reduxForm({
    form: 'newTxConsumer',
    fields,
  })(New)
)
