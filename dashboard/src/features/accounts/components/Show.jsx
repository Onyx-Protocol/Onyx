import React from 'react'
import {
  BaseShow,
  CopyableBlock,
  KeyValueTable,
  PageContent,
  PageTitle,
  RawJsonButton,
} from 'features/shared/components'

class Show extends BaseShow {
  constructor(props) {
    super(props)

    this.createControlProgram = this.createControlProgram.bind(this)
  }

  createControlProgram() {
    this.props.createControlProgram([{
      type: 'account',
      params: { account_id: this.props.item.id }
    }]).then((program) => this.props.showControlProgram(<div>
      <p>Copy this one-time use control program to use in a transaction:</p>
      <CopyableBlock value={program.control_program} />
    </div>))
  }

  render() {
    const item = this.props.item

    let view
    if (item) {
      const title = <span>
        {'Account '}
        <code>{item.alias ? item.alias : item.id}</code>
      </span>

      view = <div>
        <PageTitle
          title={title}
          actions={[
            <button className='btn btn-link' onClick={this.createControlProgram}>
              Create control program
            </button>
          ]}
        />

        <PageContent>
          <KeyValueTable
            title='Details'
            actions={[
              <button key='show-txs' className='btn btn-link' onClick={this.props.showTransactions.bind(this, item)}>Transactions</button>,
              <button key='show-balances' className='btn btn-link' onClick={this.props.showBalances.bind(this, item)}>Balances</button>,
              <RawJsonButton key='raw-json' item={item} />
            ]}
            items={[
              {label: 'ID', value: item.id},
              {label: 'Alias', value: item.alias},
              {label: 'Tags', value: item.tags},
              {label: 'Keys', value: item.keys.length},
              {label: 'Quorum', value: item.quorum},
            ]}
          />

          {item.keys.map((key, index) =>
            <KeyValueTable
              key={index}
              title={`Key ${index + 1}`}
              items={[
                {label: 'Root Xpub', value: key.root_xpub},
                {label: 'Account Xpub', value: key.account_xpub},
                {label: 'Account Derivation Path', value: key.account_derivation_path},
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
  createControlProgram: (data) => dispatch(actions.account.createControlProgram(data)),
  showControlProgram: (body) => dispatch(actions.app.showModal(
    body,
    actions.app.hideModal()
  )),
})

export default connect(
  mapStateToProps,
  mapDispatchToProps
)(Show)
