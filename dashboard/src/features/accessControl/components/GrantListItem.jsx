import React from 'react'
import PropTypes from 'prop-types'

class GrantListItem extends React.Component {
  render() {
    const item = this.props.item

    let desc
    if (item.guardType == 'access_token') {
      desc = item.guardData.id
    } else {
      desc = <div>
        {Object.keys(item.guardData).map(field => <p>
          {field}:
          <ul>
            {Object.keys(item.guardData[field]).map(key => <li>
              {key}: {item.guardData[field][key]}
            </li>)}
          </ul>
        </p>)}
      </div>
    }
    return(
      <tr>
        <td>{desc}</td>
        <td>{item.policy}</td>
        <td>
          <button className='btn btn-danger btn-xs' onClick={this.props.delete.bind(this, item)}>
            Delete
          </button>
        </td>
      </tr>
    )
  }
}

GrantListItem.propTypes = {
  item: PropTypes.object,
  delete: PropTypes.func,
}

export default GrantListItem
