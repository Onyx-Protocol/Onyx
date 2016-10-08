import React from 'react'

class ListItem extends React.Component {
  render() {
    const item = {...this.props.item}
    const label = `Consumer ${item.alias || item.id}`

    const after = item.after.split('-')[0].split(':')
    const blockHeight = after[0]
    const blockPosition = after[1]

    // int max (2147483647) is used to indicate that a consumer
    // hasn't yet been read from.
    const hasStarted = blockPosition != '2147483647'

    return(
      <div className='panel panel-default'>
        <div className='panel-heading'>
          {label}

          <button className='btn btn-danger btn-sm pull-right' onClick={this.props.delete.bind(this, item.id)}>
            <span className='glyphicon glyphicon-trash' />&nbsp;
            Delete
          </button>
        </div>
        <div className='panel-body'>
          <label>Filter</label>
          <pre>
            {item.filter}
          </pre>

          <label>Last Item</label>
          {hasStarted && <p>
            Block Height: {blockHeight}
            <br/>
            Block Position: {blockPosition}
          </p>}

          {!hasStarted && <p>
            Not yet queried
          </p>}
        </div>
      </div>
    )
  }
}

export default ListItem
