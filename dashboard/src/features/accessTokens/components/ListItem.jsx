import React from 'react'

export default class ListItem extends React.Component {
  render() {
    const item = this.props.item
    return(<tr>
      <td>
        <code>{item.id}</code>
      </td>
      <td>
        <button className='btn btn-danger' onClick={this.props.delete.bind(this, item)}>
          Delete
        </button>
      </td>
    </tr>)
  }
}
