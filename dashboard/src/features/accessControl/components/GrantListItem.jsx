import React from 'react'
import PropTypes from 'prop-types'
import { isAccessToken, getPolicyNamesString } from 'features/accessControl/selectors'
import EditPolicies from './EditPolicies'

class GrantListItem extends React.Component {
  render() {
    const item = this.props.item

    let desc
    if (isAccessToken(item)) {
      desc = item.guardData.id
    } else {
      desc = <div>
        {Object.keys(item.guardData).map(field =>
          <div key={field}>
            {field}:
            <ul>
              {Object.keys(item.guardData[field]).map(key =>
                <li key={key}>
                  {key}: {item.guardData[field][key]}
                </li>
              )}
            </ul>
          </div>
        )}
      </div>
    }
    return(
      <tr>
        <td>{desc}</td>
        {!item.isEditing && <td>
          {getPolicyNamesString(item)}
        </td>}
        {!item.isEditing && <td>
          <button className='btn btn-link' onClick={this.props.beginEditing.bind(this, item.id)}>
            Edit
          </button>

          {isAccessToken(item) && <button className='btn btn-link' onClick={this.props.delete.bind(this, item)}>
            Delete
          </button>}
        </td>}
        {item.isEditing && <td colSpan='2'>
          <EditPolicies item={item}/>
        </td>}
      </tr>
    )
  }
}

GrantListItem.propTypes = {
  item: PropTypes.object,
  delete: PropTypes.func,
}

export default GrantListItem
