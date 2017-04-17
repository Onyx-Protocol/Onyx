import React from 'react'
import PropTypes from 'prop-types'
import { isAccessToken, getPolicyNamesString } from 'features/accessControl/selectors'

class GrantListItem extends React.Component {
  render() {
    const item = this.props.item

    let desc
    if (isAccessToken(item)) {
      desc = item.guardData.id
    } else {
      desc = <div>
        {Object.keys(item.guardData).map(field => <div key={field}>
          {field}:
          <ul>
            {Object.keys(item.guardData[field]).map(key => <li key={key}>
              {key}: {item.guardData[field][key]}
            </li>)}
          </ul>
        </div>)}
      </div>
    }
    return(
      <tr>
        <td>{desc}</td>
        <td>{getPolicyNamesString(item)}</td>
        <td>
          <button className='btn btn-link btn-sm' onClick={this.props.showEdit.bind(this, item)}>
            Edit
          </button>

          {isAccessToken(item) && <button className='btn btn-link btn-sm' onClick={this.props.delete.bind(this, item)}>
            Delete
          </button>}
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
