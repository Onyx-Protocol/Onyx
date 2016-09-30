import React from 'react'
import PageHeader from '../../../components/PageHeader/PageHeader'
import { JsonField, ErrorBanner } from '../../../components/Common'
import ActionItem from './FormActionItem'
import { reduxForm } from 'redux-form'

class Form extends React.Component {
  constructor(props) {
    super(props)
    this.state = {fields: []}

    this.submitWithValidation = this.submitWithValidation.bind(this)
    this.addActionItem = this.addActionItem.bind(this)
    this.removeActionItem = this.removeActionItem.bind(this)

    this.addActionItem()
    this.addActionItem()
  }

  addActionItem() {
    this.props.fields.actions.addField({reference_data: '{\n\t\n}'})
  }

  removeActionItem() {
    this.props.fields.actions.removeField()
  }

  submitWithValidation(data) {
    return new Promise((resolve, reject) => {
      this.props.submitForm(data)
        .catch((err) => reject({_error: err.message}))
    })
  }

  render() {
    const {
      fields: { actions },
      error,
      handleSubmit,
      submitting
    } = this.props

    return(
      <div className='form-container'>
        <PageHeader title='New Transaction' />

        <form onSubmit={handleSubmit(this.submitWithValidation)} >
          <div className='form-group'>

            {!actions.length && <div className='well'>Add actions to build a transaction</div>}

            {actions.map((action, index) =>
              <ActionItem
                key={index}
                index={index}
                fieldProps={action}
                accounts={this.props.accounts}
                assets={this.props.assets}
              />)}

            <button type='button' className='btn btn-link' onClick={this.addActionItem} >
              + Add Action
            </button>

            {actions.length > 0 &&
              <button type='button' className='btn btn-link' onClick={this.removeActionItem}>
                - Remove Action
              </button>
            }
          </div>

          <hr />

          <p>
            Submitting builds a transaction template, signs the template with
             the Mock HSM, and submits the fully signed template to the blockchain.
          </p>

          {error && <ErrorBanner
            title='There was a problem submitting your transaction:'
            message={error}/>}

          <button type='submit' className='btn btn-primary' disabled={submitting}>
            Submit Transaction
          </button>
        </form>
      </div>
    )
  }
}

const validate = values => {
  const errors = {actions: {}}
  let fieldError

  values.actions.forEach((action, index) => {
    fieldError = JsonField.validator(values.actions[index].reference_data)
    if (fieldError) {
      errors.actions[index] = {...errors.actions[index], reference_data: fieldError}
    }
  })

  return errors
}

export default reduxForm({
  form: 'NewTransactionForm',
  fields: [
    'actions[].type',
    'actions[].account_id',
    'actions[].account_alias',
    'actions[].asset_id',
    'actions[].asset_alias',
    'actions[].amount',
    'actions[].control_program',
    'actions[].transaction_id',
    'actions[].position',
    'actions[].reference_data',
  ],
  validate
})(Form)
