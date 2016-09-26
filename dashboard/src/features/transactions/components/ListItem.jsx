import React from 'react'
import { Link } from 'react-router'

class ListItem extends React.Component {
  render() {
    const item = this.props.item

    return(
      <div className='panel panel-default'>
        <div className='panel-heading'>
          <strong>Transaction - {item.id}</strong>
        </div>
        <div className='panel-body'>
          <Link to={`/transactions/${item.id}`}>
            View Transaction →
          </Link>
        </div>
      </div>
    )
  }
}

export default ListItem
