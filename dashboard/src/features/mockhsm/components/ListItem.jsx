import React from 'react'
import componentClassNames from 'utility/componentClassNames'

class ListItem extends React.Component {
  render() {
    const item = this.props.item

    return(
      <tr className={componentClassNames(this)}>
        <td>{item.alias}</td>
        <td><code>{item.xpub}</code></td>
        <td></td>
      </tr>
    )
  }
}

export default ListItem
