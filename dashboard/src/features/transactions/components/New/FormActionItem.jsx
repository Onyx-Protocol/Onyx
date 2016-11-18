import React from 'react'
import {
  TextField,
  JsonField,
  ObjectSelectorField,
  Autocomplete,
} from 'components/Common'
import { ErrorBanner, HiddenField } from 'features/shared/components'
import styles from './FormActionItem.scss'

const ISSUE_KEY = 'issue'
const SPEND_ACCOUNT_KEY = 'spend_account'
const SPEND_UNSPENT_KEY = 'spend_account_unspent_output'
const CONTROL_ACCOUNT_KEY = 'control_account'
const CONTROL_PROGRAM_KEY = 'control_program'
const RETIRE_ASSET_KEY = 'retire_asset'
const TRANSACTION_REFERENCE_DATA = 'set_transaction_reference_data'

const actionLabels = {
  [ISSUE_KEY]: 'Issue',
  [SPEND_ACCOUNT_KEY]: 'Spend from account',
  [SPEND_UNSPENT_KEY]: 'Spend unspent output',
  [CONTROL_ACCOUNT_KEY]: 'Control with account',
  [CONTROL_PROGRAM_KEY]: 'Control with program',
  [RETIRE_ASSET_KEY]: 'Retire',
  [TRANSACTION_REFERENCE_DATA]: 'Set transaction reference data',
}

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
    this.state = {
      referenceDataOpen: props.fieldProps.type.value == TRANSACTION_REFERENCE_DATA
    }
    this.openReferenceData = this.openReferenceData.bind(this)
  }

  openReferenceData() {
    this.setState({referenceDataOpen: true})
  }

  componentDidMount() {
    window.scroll(
      window.scrollX,
      window.scrollY + this.scrollRef.getBoundingClientRect().top - 10
    )
  }

  render() {
    const {
      type,
      account_id,
      account_alias,
      control_program,
      transaction_id,
      position,
      asset_id,
      asset_alias,
      amount,
      reference_data } = this.props.fieldProps

    const visible = visibleFields[type.value] || {}
    const remove = (event) => {
      event.preventDefault()
      this.props.remove(this.props.index)
    }

    const classNames = [styles.main]
    if (type.error) classNames.push(styles.error)

    return (
      <div className={classNames.join(' ')} ref={ref => this.scrollRef = ref}>
        <HiddenField fieldProps={type} />

        <div className={styles.header}>
          <label className={styles.title}>{actionLabels[type.value]}</label>
          <a href='#' className='btn btn-sm btn-danger' onClick={remove}>Remove</a>
        </div>

        {type.error && <ErrorBanner message={type.error} />}

        {visible.account &&
          <ObjectSelectorField
            title='Account'
            aliasField={Autocomplete.AccountAlias}
            fieldProps={{
              id: account_id,
              alias: account_alias
            }}
          />}

        {visible.control_program &&
          <TextField title='Control Program' fieldProps={control_program} />}

        {visible.transaction_id &&
          <TextField title='Transaction ID' fieldProps={transaction_id} />}

        {visible.position &&
          <TextField title='Transaction Unspent Position' fieldProps={position} />}

        {visible.asset &&
          <ObjectSelectorField
            title='Asset'
            aliasField={Autocomplete.AssetAlias}
            fieldProps={{
              id: asset_id,
              alias: asset_alias
            }}
          />}

        {visible.amount &&
          <TextField title='Amount' fieldProps={amount} />}

        {this.state.referenceDataOpen &&
          <JsonField title='Reference data' fieldProps={reference_data} />
        }
        {!this.state.referenceDataOpen &&
          <button type='button' className='btn btn-link' onClick={this.openReferenceData}>
            Add reference data
          </button>
        }
      </div>
    )
  }
}
