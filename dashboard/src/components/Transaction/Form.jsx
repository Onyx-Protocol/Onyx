import React from 'react'
import PageHeader from "../PageHeader/PageHeader"
import { Panel, TextField, NumberField, SelectField, ErrorBanner } from "../Common"

const ISSUE_KEY = "issue"
const SPEND_ACCOUNT_KEY = "spend_account_unspent_output_selector"
const SPEND_UNSPENT_KEY = "spend_account_unspent_output"
const CONTROL_ACCOUNT_KEY = "control_account"
const CONTROL_PROGRAM_KEY = "control_program"
const RETIRE_ASSET_KEY = "retire_asset"

const actionTypes = {}
actionTypes[ISSUE_KEY] = "Issue"
actionTypes[SPEND_ACCOUNT_KEY] = "Spend from Account"
actionTypes[SPEND_UNSPENT_KEY] = "Spend from Unspent Output"
actionTypes[CONTROL_ACCOUNT_KEY] = "Control with Account"
actionTypes[CONTROL_PROGRAM_KEY] = "Control with Program"
actionTypes[RETIRE_ASSET_KEY] = "Retire"

class Form extends React.Component {
  constructor(props) {
    super(props)
    this.state = {fields: []}

    this.showField = this.showField.bind(this)
    this.buildFieldsForAction = this.selectFieldsForAction.bind(this)
    this.submitWithValidation = this.submitWithValidation.bind(this)

    // Add single initial action
    this.props.fields.actions.addField()
    this.props.fields.actions.addField()
  }

  selectFieldsForAction(type, action, index) {
    let fields = this.state.fields.slice()
    switch (type) {
    case ISSUE_KEY:
      fields[index] = {asset_alias: true, amount: true}
      break
    case SPEND_ACCOUNT_KEY:
      fields[index] = {asset_alias: true, account_alias: true, amount: true}
      break
    case SPEND_UNSPENT_KEY:
      fields[index] = {transaction_id: true, position: true}
      break
    case CONTROL_ACCOUNT_KEY:
      fields[index] = {asset_alias: true, account_alias: true, amount: true}
      break
    case CONTROL_PROGRAM_KEY:
      fields[index] = {asset_alias: true, control_program: true, amount: true}
      break
    case RETIRE_ASSET_KEY:
      fields[index] = {asset_alias: true, amount: true}
      break
    default:
      fields[index] = {}
    }

    this.setState({fields})
  }

  showField(fieldName, index) {
    let field = this.state.fields[index]
    return field ? field[fieldName] : false
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
      <div className="form-container">
        <PageHeader title="New Transaction" />

        <form onSubmit={handleSubmit(this.submitWithValidation)} >
          <div className='form-group'>

            {!actions.length && <div className='well'>Add actions to build a transaction</div>}
            {actions.map((action, index) => {
              let typeOnChange = (event) => {
                let result = action.type.onChange(event)
                this.selectFieldsForAction(result.value, action, index)
              }
              let typeProps = Object.assign({}, action.type, {onChange: typeOnChange})

              return (
                <Panel title={`Action ${index + 1}`} key={index}>
                  <SelectField title="Type" emptyLabel="Select an action type..." options={actionTypes} fieldProps={typeProps} />

                  {this.showField('asset_alias', index) &&
                    <TextField title="Asset Alias" fieldProps={action.params.asset_alias} />}

                  {this.showField('account_alias', index) &&
                    <TextField title="Account Alias" fieldProps={action.params.account_alias} />}

                  {this.showField('control_program', index) &&
                    <TextField title="Control Program" fieldProps={action.params.control_program} />}

                  {this.showField('amount', index) &&
                    <NumberField title="Amount" fieldProps={action.params.amount} />}

                  {this.showField('transaction_id', index) &&
                    <TextField title="Transaction ID" fieldProps={action.params.transaction_id} />}

                  {this.showField('position', index) &&
                    <NumberField title="Transaction Unspent Position" fieldProps={action.params.position} />}

                </Panel>
              )
            })}

            <button type="button" className="btn btn-link" onClick={() => actions.addField()} >
              + Add Action
            </button>

            {actions.length > 0 && <button type="button" className="btn btn-link" onClick={() => {
              actions.removeField()
            }}>
              - Remove Action
            </button>}
          </div>

          <hr />

          <p>
            Submitting builds a transaction template, signs the template with
             the Mock HSM, and submits the fully signed template to the blockchain.
          </p>

          {error && <ErrorBanner
            title="There was a problem submitting your transaction:"
            message={error}/>}

          <button type="submit" className="btn btn-primary" disabled={submitting}>Submit Transaction</button>
        </form>
      </div>
    )
  }
}

export default Form
