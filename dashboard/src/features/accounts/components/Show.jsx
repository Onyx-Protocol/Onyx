import React from 'react'
import {
  BaseShow,
  PageTitle,
  Table,
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
      <pre>{program.control_program}</pre>
    </div>))
  }

  render() {
    const item = this.props.item

    let view
    if (item) {
      const title = <span>
        {'Account '}
        <code>{item.alias ? item.alias :item.id}</code>
      </span>

      view = <div>
        <PageTitle
          title={title}
          actions={
            <button className='btn btn-link' onClick={this.createControlProgram}>
              Create Control Program
            </button>}
        />

        <Table
          title='details'
          actions={[
            <button className='btn btn-link' onClick={this.props.showTransactions.bind(this, item.id)}>Transactions</button>,
            <button className='btn btn-link' onClick={this.props.showBalances.bind(this, item.id)}>Balances</button>
          ]}
          items={[
            {label: 'ID', value: item.id},
            {label: 'Alias', value: item.alias},
            {label: 'Tags', value: item.tags},
            {label: 'Quorum', value: item.quorum},
          ]}
        />

        {item.keys.map((key, index) =>
          <Table
            key={index}
            title={index == 0 ? 'Keys' : ''}
            items={[
              {label: 'Root Xpub', value: key.root_xpub},
              {label: 'Account Xpub', value: key.account_xpub},
              {label: 'Account Derivation Path', value: key.account_derivation_path},
            ]}
          />
        )}
      </div>
    }
    return this.renderIfFound(view)
  }
}

// Container

import { connect } from 'react-redux'
import { push } from 'react-router-redux'
import actions from 'actions'

const mapStateToProps = (state, ownProps) => ({
  item: state.account.items[ownProps.params.id]
})

const mapDispatchToProps = ( dispatch ) => ({
  fetchItem: (id) => dispatch(actions.account.fetchItems({filter: `id='${id}'`})),
  showTransactions: (id) => {
    let query = `inputs(account_id='${id}') OR outputs(account_id='${id}')`
    dispatch(actions.transaction.updateQuery(query))
    dispatch(push('/transactions'))
  },
  showBalances: (id) => {
    let query = `account_id='${id}'`
    dispatch(actions.balance.updateQuery({
      query: query,
      sumBy: 'asset_id'
    }))
    dispatch(push('/balances'))
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
