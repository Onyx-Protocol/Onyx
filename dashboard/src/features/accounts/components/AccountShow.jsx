import React from 'react'
import {
  BaseShow,
  KeyValueTable,
  PageContent,
  PageTitle,
  RawJsonButton,
} from 'features/shared/components'
import ReceiverModal from './ReceiverModal'
import componentClassNames from 'utility/componentClassNames'

class AccountShow extends BaseShow {
  constructor(props) {
    super(props)

    this.createReceiver = this.createReceiver.bind(this)
  }

  createReceiver() {
    this.props.createReceiver({
      accountId: this.props.item.id
    }).then((rec) => this.props.showReceiver(<ReceiverModal receiver={rec}/>))
  }

  render() {
    const item = this.props.item

    let view
    if (item) {
      const title = <span>
        {'Account '}
        <code>{item.alias ? item.alias : item.id}</code>
      </span>

      view = <div className={componentClassNames(this)}>
        <PageTitle
          title={title}
          actions={[
            <button className='btn btn-link' onClick={this.createReceiver}>
              Create receiver
            </button>
          ]}
        />

        <PageContent>
          <KeyValueTable
            id={item.id}
            object='account'
            title='Details'
            actions={[
              <button key='show-txs' className='btn btn-link' onClick={this.props.showTransactions.bind(this, item)}>Transactions</button>,
              <button key='show-balances' className='btn btn-link' onClick={this.props.showBalances.bind(this, item)}>Balances</button>,
              <RawJsonButton key='raw-json' item={item} />
            ]}
            items={[
              {label: 'ID', value: item.id},
              {label: 'Alias', value: item.alias},
              {label: 'Tags', value: item.tags, editUrl: `/accounts/${item.id}/tags`},
              {label: 'Keys', value: item.keys.length},
              {label: 'Quorum', value: item.quorum},
            ]}
          />

          {item.keys.map((key, index) =>
            <KeyValueTable
              key={index}
              title={`Key ${index + 1}`}
              items={[
                {label: 'Root Xpub', value: key.rootXpub},
                {label: 'Account Xpub', value: key.accountXpub},
                {label: 'Account Derivation Path', value: key.accountDerivationPath},
              ]}
            />
          )}
        </PageContent>
      </div>
    }
    return this.renderIfFound(view)
  }
}

// Container

import { connect } from 'react-redux'
import actions from 'actions'

const mapStateToProps = (state, ownProps) => ({
  item: state.account.items[ownProps.params.id]
})

const mapDispatchToProps = ( dispatch ) => ({
  fetchItem: (id) => dispatch(actions.account.fetchItems({filter: `id='${id}'`})),
  showTransactions: (item) => {
    let filter = `inputs(account_id='${item.id}') OR outputs(account_id='${item.id}')`
    if (item.alias) {
      filter = `inputs(account_alias='${item.alias}') OR outputs(account_alias='${item.alias}')`
    }

    dispatch(actions.transaction.pushList({ filter }))
  },
  showBalances: (item) => {
    let filter = `account_id='${item.id}'`
    if (item.alias) {
      filter = `account_alias='${item.alias}'`
    }

    dispatch(actions.balance.pushList({ filter }))
  },
  createReceiver: (data) => dispatch(actions.account.createReceiver(data)),
  showReceiver: (body) => dispatch(actions.app.showModal(
    'New account receiver created',
    body,
  ))
})

export default connect(
  mapStateToProps,
  mapDispatchToProps
)(AccountShow)
