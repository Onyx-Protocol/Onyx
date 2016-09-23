import React from 'react'
import {
  Panel,
  TextField,
  NumberField,
  SelectField,
  JsonField,
  AccountField,
  AssetField
} from '../Common'

const ISSUE_KEY = 'issue'
const SPEND_ACCOUNT_KEY = 'spent_account'
const SPEND_UNSPENT_KEY = 'spend_account_unspent_output'
const CONTROL_ACCOUNT_KEY = 'control_account'
const CONTROL_PROGRAM_KEY = 'control_program'
const RETIRE_ASSET_KEY = 'retire_asset'

const actionTypes = [
  {value: ISSUE_KEY, label: 'Issue'},
  {value: SPEND_ACCOUNT_KEY, label: 'Spend from Account'},
  {value: SPEND_UNSPENT_KEY, label: 'Spend Unspent Output'},
  {value: CONTROL_ACCOUNT_KEY, label: 'Control with Account'},
  {value: CONTROL_PROGRAM_KEY, label: 'Control with Program'},
  {value: RETIRE_ASSET_KEY, label: 'Retire'}
]

const visibleFields = {
  [ISSUE_KEY]: {asset_alias: true, amount: true},
  [SPEND_ACCOUNT_KEY]: {asset_alias: true, account_alias: true, amount: true},
  [SPEND_UNSPENT_KEY]: {transaction_id: true, position: true},
  [CONTROL_ACCOUNT_KEY]: {asset_alias: true, account_alias: true, amount: true},
  [CONTROL_PROGRAM_KEY]: {asset_alias: true, control_program: true, amount: true},
  [RETIRE_ASSET_KEY]: {asset_alias: true, amount: true},
}

export default class ActionItem extends React.Component {
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

        <SelectField title='Type' emptyLabel='Select an action type...' options={actionTypes} fieldProps={typeProps} />

        {visible.account_alias &&
          <AccountField
            title='Account Alias'
            fieldProps={this.props.fieldProps.account_alias}
          />}

        {visible.control_program &&
          <TextField title='Control Program' fieldProps={this.props.fieldProps.control_program} />}

        {visible.transaction_id &&
          <TextField title='Transaction ID' fieldProps={this.props.fieldProps.transaction_id} />}

        {visible.position &&
          <NumberField title='Transaction Unspent Position' fieldProps={this.props.fieldProps.position} />}

        {visible.asset_alias &&
          <AssetField
            title='Asset Alias'
            fieldProps={this.props.fieldProps.asset_alias}
          />}

        {visible.amount &&
          <NumberField title='Amount' fieldProps={this.props.fieldProps.amount} />}

        {this.state.selectedType && this.state.referenceDataOpen &&
          <JsonField title='Reference data' fieldProps={this.props.fieldProps.reference_data} />
        }
        {this.state.selectedType && !this.state.referenceDataOpen &&
          <button type='button' className='btn btn-link' onClick={this.openReferenceData}>
            Add reference data
          </button>
        }

      </Panel>
    )
  }
}
