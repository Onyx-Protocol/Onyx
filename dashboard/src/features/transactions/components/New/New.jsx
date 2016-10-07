import { BaseNew, FormContainer } from 'features/shared/components'
import { TextField, JsonField } from 'components/Common'
import { reduxForm } from 'redux-form'
import ActionItem from './FormActionItem'
import React from 'react'
import styles from './New.scss'

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
      fields: { base_transaction, actions, submit_action },
      error,
      handleSubmit,
      submitting
    } = this.props

    return(
      <FormContainer
        error={error}
        label='New Transaction'
        onSubmit={handleSubmit(this.submitWithValidation)}
        submitting={submitting} >

        {this.state.showBaseTx &&
          <TextField
            title='Base transaction:'
            placeholder='Paste transaction hex here...'
            fieldProps={base_transaction}
            autoFocus={true}
          />
        }
        {!this.state.showBaseTx &&
          <button
            type='button'
            className='btn btn-link'
            onClick={() => this.setState({showBaseTx: true})
          }>
            Include a base transaction
          </button>
        }

        <hr />

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

        <table className={styles.submitTable}>
          <tbody>
            <tr>
              <td><input id='submit_action_submit' type='radio' {...submit_action} value='submit' checked={submit_action.value == 'submit'} /></td>
              <td><label htmlFor='submit_action_submit'>Submit transaction to blockchain</label></td>
            </tr>
            <tr>
              <td></td>
              <td><label htmlFor='submit_action_submit' className={styles.submitDescription}>
                This transaction will be signed by the Mock HSM and submitted to the blockchain.
              </label></td>
            </tr>
            <tr>
              <td><input id='submit_action_generate' type='radio' {...submit_action} value='generate' checked={submit_action.value == 'generate'} /></td>
              <td><label htmlFor='submit_action_generate'>Allow additional actions</label></td>
            </tr>
            <tr>
              <td></td>
              <td><label htmlFor='submit_action_generate' className={styles.submitDescription}>
                These actions will be signed by the Mock HSM and returned as a transaction hex string, which should be used as the base transaction in a multi-party swap.
              </label></td>
            </tr>
          </tbody>
        </table>

        <hr />
      </FormContainer>
    )
  }
}

const validate = values => {
  const errors = {actions: {}}

  // Base transaction
  let baseTx = values.base_transaction || ''
  if (baseTx.trim().match(/[^0-9a-fA-F]/)) {
    errors.base_transaction = 'Base transaction must be a hex string.'
  }

  // Actions
  let fieldError
  values.actions.forEach((action, index) => {
    fieldError = JsonField.validator(values.actions[index].reference_data)
    if (fieldError) {
      errors.actions[index] = {...errors.actions[index], reference_data: fieldError}
    }
  })

  return errors
}

export default BaseNew.connect(
  BaseNew.mapStateToProps('transaction'),
  BaseNew.mapDispatchToProps('transaction'),
  reduxForm({
    form: 'NewTransactionForm',
    fields: [
      'base_transaction',
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
      'submit_action',
    ],
    validate,
    initialValues: {
      submit_action: 'submit',
    },
  }
  )(Form)
)
