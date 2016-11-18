import React from 'react'
import { Link } from 'react-router'

class ListItem extends React.Component {
  render() {
    const item = this.props.item

    return(
      <tr>
        <td>{item.alias || '-'}</td>
        <td><code>{item.id}</code></td>
        <td>
          <Link to={`/assets/${item.id}`}>
            View details →
          </Link>
        </td>
      </tr>
    )
  }
}

export default ListItem
