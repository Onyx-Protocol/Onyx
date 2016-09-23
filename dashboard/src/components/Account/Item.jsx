import React from 'react'
import styles from './Item.scss'

class Item extends React.Component {
  constructor(props) {
    super(props)

    this.state = {}
  }

  createControlProgram() {
    this.props.createControlProgram([{
      type: "account",
      params: { account_id: this.props.item.id }
    }]).then((program) => {
      this.setState({program: program.control_program})
    })
  }

  render() {
    const item = this.props.item
    const title = item.alias ?
      `Account - ${item.alias}` :
      `Account - ${item.id}`

    return(
      <div className="panel panel-default">
        <div className="panel-heading">
          <strong>{title}</strong>
        </div>
        <div className="panel-body">
          <pre>
            {JSON.stringify(item, null, '  ')}
          </pre>
        </div>
        <div className="panel-footer">
          <div className="row">
            <div className="col-sm-4">
              <ul className="nav nav-pills">
                <li>
                  <button className="btn btn-link" onClick={this.props.showTransactions.bind(this, item.id)}>Transactions</button>
                </li>
                <li>
                  <button className="btn btn-link" onClick={this.props.showBalances.bind(this, item.id)}>Balances</button>
                </li>
              </ul>
            </div>
            <div className="col-sm-8 text-right">
              <button className="btn btn-link" onClick={this.createControlProgram.bind(this, item.id)}>
                Create&nbsp;
                {this.state.program && "another "}
                Control Program
              </button>
              {this.state.program && <p>
                <code className={styles.control_program}>{this.state.program}</code>
              </p>}
            </div>
          </div>
        </div>
      </div>
    )
  }
}

export default Item
