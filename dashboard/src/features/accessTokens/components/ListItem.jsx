import React from 'react'

export default class ListItem extends React.Component {
  render() {
    const item = this.props.item
    return(<div className='panel panel-default'>
      <div className='panel-heading'>
        {item.id}
      </div>

      <div className='panel-body'>
        <button className='btn btn-danger' onClick={this.props.delete.bind(this, item.id)}>
          Delete
        </button>
      </div>
    </div>)
  }
}
