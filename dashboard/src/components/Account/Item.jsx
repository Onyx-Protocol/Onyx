import React from 'react';

class Item extends React.Component {
  render() {
    const item = this.props.item
    return(
      <div className="panel panel-default">
        <div className="panel-heading">
          <strong>Account {item.id}</strong>
        </div>
        <div className="panel-body">
          <pre>
            {JSON.stringify(item, null, '  ')}
          </pre>
        </div>
        <div className="panel-footer">
          <ul className="nav nav-pills">
            <li>
              <a onClick={this.props.showTransactions.bind(this, item.id)}>Transactions</a>
            </li>
            <li>
              <a onClick={this.props.showBalances.bind(this, item.id)}>Balances</a>
            </li>
          </ul>
        </div>
      </div>
    )
  }
}

export default Item
