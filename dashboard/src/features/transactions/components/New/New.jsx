import { BaseNew, FormContainer, FormSection, FieldLabel } from 'features/shared/components'
import { TextField, JsonField } from 'components/Common'
import { DropdownButton, MenuItem } from 'react-bootstrap'
import { reduxForm } from 'redux-form'
import ActionItem from './FormActionItem'
import React from 'react'
import styles from './New.scss'

class Form extends React.Component {
  constructor(props) {
    super(props)
    this.state = {
      showDropdown: false
    }

    this.submitWithValidation = this.submitWithValidation.bind(this)
    this.addActionItem = this.addActionItem.bind(this)
    this.removeActionItem = this.removeActionItem.bind(this)
    this.toggleDropwdown = this.toggleDropwdown.bind(this)
    this.closeDropdown = this.closeDropdown.bind(this)
  }

  toggleDropwdown() {
    this.setState({ showDropdown: !this.state.showDropdown })
  }

  closeDropdown() {
    this.setState({ showDropdown: false })
  }

  addActionItem(type) {
    this.props.fields.actions.addField({
      type: type,
      reference_data: '{\n\t\n}'
    })
    this.closeDropdown()
  }

  removeActionItem(index) {
    this.props.fields.actions.removeField(index)
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
        submitLabel='Submit Transaction'
        onSubmit={handleSubmit(this.submitWithValidation)}
        submitting={submitting} >

        <FormSection title='Actions'>
          {actions.map((action, index) =>
            <ActionItem
              key={index}
              index={index}
              fieldProps={action}
              accounts={this.props.accounts}
              assets={this.props.assets}
              remove={this.removeActionItem}
            />)}

            <div className={`btn-group ${this.state.showDropdown && 'open'}`}>
              <DropdownButton
                className={`btn-primary ${styles.addAction}`}
                id='input-dropdown-addon'
                title='+ Add Action'
                onSelect={this.addActionItem}
              >
                <MenuItem eventKey='issue'>Issue</MenuItem>
                <MenuItem eventKey='spend_account'>Spend from Account</MenuItem>
                <MenuItem eventKey='spend_account_unspent_output'>Spend Unspent Output</MenuItem>
                <MenuItem eventKey='control_account'>Control with Account</MenuItem>
                <MenuItem eventKey='control_program'>Control with Program</MenuItem>
                <MenuItem eventKey='retire_asset'>Retire</MenuItem>
                <MenuItem eventKey='set_transaction_reference_data'>Set Transaction Reference Data</MenuItem>
              </DropdownButton>
            </div>
        </FormSection>

        <FormSection title='Advanced Options'>
          {!this.state.showAdvanced &&
            <a href='#'
              onClick={(e) => {
                e.preventDefault()
                this.setState({showAdvanced: true})
              }}
            >
              Show advanced options
            </a>
          }
          {this.state.showAdvanced &&
            <div>
              <TextField
                title='Base transaction'
                placeholder='Paste transaction hex here...'
                fieldProps={base_transaction}
                autoFocus={true} />

              <FieldLabel>Transaction Build Type</FieldLabel>
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
                      These actions will be signed by the Mock HSM and returned as a
                      transaction hex string, which should be used as the base
                      transaction in a multi-party swap. This transaction will be
                      valid for one hour.
                    </label></td>
                  </tr>
                </tbody>
              </table>
            </div>
          }
        </FormSection>
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
