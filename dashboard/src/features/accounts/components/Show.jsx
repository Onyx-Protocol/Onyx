import React from 'react'
import { BaseShow } from '../../shared'

class Show extends BaseShow {
  constructor(props) {
    super(props)

    this.createControlProgram = this.createControlProgram.bind(this)
  }

  createControlProgram() {
    this.props.createControlProgram([{
      type: 'account',
      params: { account_id: this.props.item.id }
    }]).then((program) => {
      this.setState({program: program.control_program})
    })
  }

  render() {
    const item = this.props.item

    let view
    if (item) {
      view = <div className='panel panel-default'>
        <div className='panel-heading'>
          <strong>Account - {item.id}</strong>
        </div>
        <div className='panel-body'>
          <pre>
            {JSON.stringify(item, null, '  ')}
          </pre>
        </div>

        <div className='panel-footer'>
          <div className='row'>
            <div className='col-sm-4'>
              <ul className='nav nav-pills'>
                <li>
                  <button className='btn btn-link' onClick={this.props.showTransactions.bind(this, item.id)}>Transactions</button>
                </li>
                <li>
                  <button className='btn btn-link' onClick={this.props.showBalances.bind(this, item.id)}>Balances</button>
                </li>
              </ul>
            </div>
            <div className='col-sm-8 text-right'>
              <button className='btn btn-link' onClick={this.createControlProgram}>
                Create&nbsp;
                {this.state.program && 'another '}
                Control Program
              </button>
              {this.state.program && <p>
                <code>{this.state.program}</code>
              </p>}
            </div>
          </div>
        </div>
      </div>
    }
    return this.renderIfFound(view)
  }
}

// Container

import { actions } from '../'
import { connect } from 'react-redux'
import { push } from 'react-router-redux'
import allActions from '../../../actions'

const mapStateToProps = (state, ownProps) => ({
  item: state.account.items[ownProps.params.id]
})

const mapDispatchToProps = ( dispatch ) => ({
  fetchItem: (id) => dispatch(actions.fetchItems({filter: `id='${id}'`})),
  showTransactions: (id) => {
    let query = `inputs(account_id='${id}') OR outputs(account_id='${id}')`
    dispatch(allActions.transaction.updateQuery(query))
    dispatch(push('/transactions'))
  },
  showBalances: (id) => {
    let query = `account_id='${id}'`
    dispatch(allActions.balance.updateQuery({
      query: query,
      sumBy: 'asset_id'
    }))
    dispatch(push('/balances'))
  },
  createControlProgram: (data) => dispatch(actions.createControlProgram(data))
})

export default connect(
  mapStateToProps,
  mapDispatchToProps
)(Show)
