import React from 'react'
import { Link } from 'react-router'

class ListItem extends React.Component {
  render() {
    const item = this.props.item
    const title = item.alias ?
      `Asset - ${item.alias}` :
      `Asset - ${item.id}`

    return(
      <div className='panel panel-default'>
        <div className='panel-heading'>
          <strong>{title}</strong>
        </div>
        <div className='panel-body'>
          <Link to={`/assets/${item.id}`}>
            View Asset →
          </Link>
        </div>
      </div>
    )
  }
}

export default ListItem
