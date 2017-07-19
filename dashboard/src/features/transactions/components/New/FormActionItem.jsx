import React from 'react'
import { ErrorBanner, HiddenField, Autocomplete, JsonField, TextField, ObjectSelectorField } from 'features/shared/components'
import componentClassNames from 'utility/componentClassNames'
import styles from './FormActionItem.scss'

const ISSUE_KEY = 'issue'
const SPEND_ACCOUNT_KEY = 'spend_account'
const SPEND_UNSPENT_KEY = 'spend_account_unspent_output'
const CONTROL_ACCOUNT_KEY = 'control_account'
const CONTROL_RECEIVER_KEY = 'control_receiver'
const RETIRE_ASSET_KEY = 'retire'
const TRANSACTION_REFERENCE_DATA = 'set_transaction_reference_data'

const actionLabels = {
  [ISSUE_KEY]: 'Issue',
  [SPEND_ACCOUNT_KEY]: 'Spend from account',
  [SPEND_UNSPENT_KEY]: 'Spend unspent output',
  [CONTROL_ACCOUNT_KEY]: 'Control with account',
  [CONTROL_RECEIVER_KEY]: 'Control with receiver',
  [RETIRE_ASSET_KEY]: 'Retire',
  [TRANSACTION_REFERENCE_DATA]: 'Set transaction reference data',
}

const visibleFields = {
  [ISSUE_KEY]: {asset: true, amount: true},
  [SPEND_ACCOUNT_KEY]: {asset: true, account: true, amount: true},
  [SPEND_UNSPENT_KEY]: {outputId: true},
  [CONTROL_ACCOUNT_KEY]: {asset: true, account: true, amount: true},
  [CONTROL_RECEIVER_KEY]: {asset: true, receiver: true, amount: true},
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
      outputId,
      type,
      accountId,
      accountAlias,
      receiver,
      assetId,
      assetAlias,
      amount,
      referenceData } = this.props.fieldProps

    const visible = visibleFields[type.value] || {}
    const remove = (event) => {
      event.preventDefault()
      this.props.remove(this.props.index)
    }

    const classNames = [styles.main]
    if (type.error) classNames.push(styles.error)

    return (
      <div className={componentClassNames(this, classNames)} ref={ref => this.scrollRef = ref}>
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
              id: accountId,
              alias: accountAlias
            }}
          />}

        {visible.receiver &&
          <JsonField title='Receiver' fieldProps={receiver} />}

        {visible.outputId &&
          <TextField title='Output ID' fieldProps={outputId} />}

        {visible.asset &&
          <ObjectSelectorField
            title='Asset'
            aliasField={Autocomplete.AssetAlias}
            fieldProps={{
              id: assetId,
              alias: assetAlias
            }}
          />}

        {visible.amount &&
          <TextField title='Amount' fieldProps={amount} />}

        {this.state.referenceDataOpen &&
          <JsonField title='Reference data' fieldProps={referenceData} />
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
