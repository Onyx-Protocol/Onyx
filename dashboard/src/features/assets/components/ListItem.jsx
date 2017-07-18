import React from 'react'
import { Link } from 'react-router'
import componentClassNames from 'utility/componentClassNames'

class ListItem extends React.Component {
  render() {
    const item = this.props.item

    return(
      <tr className={componentClassNames(this)}>
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
