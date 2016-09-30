import React from 'react'
import {
  Panel,
  TextField,
  NumberField,
  SelectField,
  JsonField,
  ObjectSelectorField,
  Autocomplete
} from '../../../components/Common'

const ISSUE_KEY = 'issue'
const SPEND_ACCOUNT_KEY = 'spend_account'
const SPEND_UNSPENT_KEY = 'spend_account_unspent_output'
const CONTROL_ACCOUNT_KEY = 'control_account'
const CONTROL_PROGRAM_KEY = 'control_program'
const RETIRE_ASSET_KEY = 'retire_asset'
const TRANSACTION_REFERENCE_DATA = 'set_transaction_reference_data'

const actionTypes = [
  {value: ISSUE_KEY, label: 'Issue'},
  {value: SPEND_ACCOUNT_KEY, label: 'Spend from Account'},
  {value: SPEND_UNSPENT_KEY, label: 'Spend Unspent Output'},
  {value: CONTROL_ACCOUNT_KEY, label: 'Control with Account'},
  {value: CONTROL_PROGRAM_KEY, label: 'Control with Program'},
  {value: RETIRE_ASSET_KEY, label: 'Retire'},
  {value: TRANSACTION_REFERENCE_DATA, label: 'Set Transaction Reference Data'},
]

const visibleFields = {
  [ISSUE_KEY]: {asset: true, amount: true},
  [SPEND_ACCOUNT_KEY]: {asset: true, account: true, amount: true},
  [SPEND_UNSPENT_KEY]: {transaction_id: true, position: true},
  [CONTROL_ACCOUNT_KEY]: {asset: true, account: true, amount: true},
  [CONTROL_PROGRAM_KEY]: {asset: true, control_program: true, amount: true},
  [RETIRE_ASSET_KEY]: {asset: true, amount: true},
  [TRANSACTION_REFERENCE_DATA]: {},
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
        selectedType: selected,
        referenceDataOpen: selected == TRANSACTION_REFERENCE_DATA
      })
    }
    let typeProps = Object.assign({}, this.props.fieldProps.type, {onChange: typeOnChange})
    let visible = visibleFields[this.state.selectedType] || {}

    return (
      <Panel title={`Action ${this.props.index + 1}`} >

        <SelectField title='Type' emptyLabel='Select an action type...' options={actionTypes} fieldProps={typeProps} />

        {visible.account &&
          <ObjectSelectorField
            title='Account'
            aliasField={Autocomplete.AccountAlias}
            fieldProps={{
              id: this.props.fieldProps.account_id,
              alias: this.props.fieldProps.account_alias
            }}
          />}

        {visible.control_program &&
          <TextField title='Control Program' fieldProps={this.props.fieldProps.control_program} />}

        {visible.transaction_id &&
          <TextField title='Transaction ID' fieldProps={this.props.fieldProps.transaction_id} />}

        {visible.position &&
          <NumberField title='Transaction Unspent Position' fieldProps={this.props.fieldProps.position} />}

        {visible.asset &&
          <ObjectSelectorField
            title='Asset'
            aliasField={Autocomplete.AssetAlias}
            fieldProps={{
              id: this.props.fieldProps.asset_id,
              alias: this.props.fieldProps.asset_alias
            }}
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
