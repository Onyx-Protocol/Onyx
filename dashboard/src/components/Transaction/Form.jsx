import React from 'react'
import PageHeader from "../PageHeader/PageHeader"
import { Panel, TextField, NumberField, SelectField, TextareaField, ErrorBanner } from "../Common"

const ISSUE_KEY = "issue"
const SPEND_ACCOUNT_KEY = "spend_account_unspent_output_selector"
const SPEND_UNSPENT_KEY = "spend_account_unspent_output"
const CONTROL_ACCOUNT_KEY = "control_account"
const CONTROL_PROGRAM_KEY = "control_program"
const RETIRE_ASSET_KEY = "retire_asset"

const actionTypes = {}
actionTypes[ISSUE_KEY] = "Issue"
actionTypes[SPEND_ACCOUNT_KEY] = "Spend from Account"
actionTypes[SPEND_UNSPENT_KEY] = "Spend Unspent Output"
actionTypes[CONTROL_ACCOUNT_KEY] = "Control with Account"
actionTypes[CONTROL_PROGRAM_KEY] = "Control with Program"
actionTypes[RETIRE_ASSET_KEY] = "Retire"

const visibleFields = {
  [ISSUE_KEY]: {asset_alias: true, amount: true},
  [SPEND_ACCOUNT_KEY]: {asset_alias: true, account_alias: true, amount: true},
  [SPEND_UNSPENT_KEY]: {transaction_id: true, position: true},
  [CONTROL_ACCOUNT_KEY]: {asset_alias: true, account_alias: true, amount: true},
  [CONTROL_PROGRAM_KEY]: {asset_alias: true, control_program: true, amount: true},
  [RETIRE_ASSET_KEY]: {asset_alias: true, amount: true},
}

class ActionItem extends React.Component {
  constructor(props) {
    super(props)
    this.state = {}
    this.openReferenceData = this.openReferenceData.bind(this)
  }

  openReferenceData() {
    this.setState({referenceDataOpen: true})
  }

  render() {
    let typeOnChange = event => {
      let selected = this.props.fieldProps.type.onChange(event).value
      this.setState({
        selectedType: selected
      })
    }
    let typeProps = Object.assign({}, this.props.fieldProps.type, {onChange: typeOnChange})
    let visible = visibleFields[this.state.selectedType] || {}

    return (
      <Panel title={`Action ${this.props.index + 1}`} >

        <SelectField title="Type" emptyLabel="Select an action type..." options={actionTypes} fieldProps={typeProps} />

        {visible.account_alias &&
          <TextField title="Account Alias" fieldProps={this.props.fieldProps.account_alias} />}

        {visible.control_program &&
          <TextField title="Control Program" fieldProps={this.props.fieldProps.control_program} />}

        {visible.transaction_id &&
          <TextField title="Transaction ID" fieldProps={this.props.fieldProps.transaction_id} />}

        {visible.position &&
          <NumberField title="Transaction Unspent Position" fieldProps={this.props.fieldProps.position} />}

        {visible.asset_alias &&
          <TextField title="Asset Alias" fieldProps={this.props.fieldProps.asset_alias} />}

        {visible.amount &&
          <NumberField title="Amount" fieldProps={this.props.fieldProps.amount} />}

        {this.state.selectedType && this.state.referenceDataOpen &&
          <TextareaField title='Reference data' fieldProps={this.props.fieldProps.reference_data} />
        }
        {this.state.selectedType && !this.state.referenceDataOpen &&
          <button type="button" className="btn btn-link" onClick={this.openReferenceData}>
            Add reference data
          </button>
        }

      </Panel>
    )
  }
}

class Form extends React.Component {
  constructor(props) {
    super(props)
    this.state = {fields: []}

    this.submitWithValidation = this.submitWithValidation.bind(this)
    this.openReferenceData = this.openReferenceData.bind(this)
    this.addActionItem = this.addActionItem.bind(this)
    this.removeActionItem = this.removeActionItem.bind(this)

    this.addActionItem()
    this.addActionItem()
  }

  addActionItem() {
    this.props.fields.actions.addField({reference_data: '{}'})
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

  openReferenceData() {
    this.setState({referenceDataOpen: true})
  }

  render() {
    const {
      fields: { actions, reference_data },
      error,
      handleSubmit,
      submitting
    } = this.props

    return(
      <div className="form-container">
        <PageHeader title="New Transaction" />

        <form onSubmit={handleSubmit(this.submitWithValidation)} >
          <div className='form-group'>

            {!actions.length && <div className='well'>Add actions to build a transaction</div>}

            {actions.map((action, index) => <ActionItem key={index} index={index} fieldProps={action} />)}

            <button type="button" className="btn btn-link" onClick={this.addActionItem} >
              + Add Action
            </button>

            {actions.length > 0 &&
              <button type="button" className="btn btn-link" onClick={this.removeActionItem}>
                - Remove Action
              </button>
            }
          </div>

          <hr />

          {this.state.referenceDataOpen &&
            <TextareaField title='Transaction-level reference data' fieldProps={reference_data} />
          }
          {!this.state.referenceDataOpen &&
            <button type="button" className="btn btn-link" onClick={this.openReferenceData}>
              Add transaction-level reference data
            </button>
          }

          <hr />

          <p>
            Submitting builds a transaction template, signs the template with
             the Mock HSM, and submits the fully signed template to the blockchain.
          </p>

          {error && <ErrorBanner
            title="There was a problem submitting your transaction:"
            message={error}/>}

          <button type="button" className="btn btn-link" type="submit" className="btn btn-primary" disabled={submitting}>
            Submit Transaction
          </button>
        </form>
      </div>
    )
  }
}

export default Form
