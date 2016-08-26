import React from 'react'

class Item extends React.Component {
  render() {
    const item = this.props.item
    let label = item.id

    if (item.alias) {
      label = item.alias
    }

    return(
      <div className="panel panel-default">
        <div className="panel-heading">
          <strong>Asset - {label}</strong>
        </div>
        <div className="panel-body">
          <pre>
            {JSON.stringify(item, null, '  ')}
          </pre>
        </div>
        <div className="panel-footer">
          <ul className="nav nav-pills">
            <li>
              <button className="btn btn-link" onClick={this.props.showCirculation.bind(this, item)}>
                Circulation
              </button>
            </li>
          </ul>
        </div>

      </div>
    )
  }
}

export default Item
