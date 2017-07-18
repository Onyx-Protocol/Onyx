import React from 'react'
import styles from './TableList.scss'
import componentClassNames from 'utility/componentClassNames'

class TableList extends React.Component {
  render() {
    return (
      <table className={componentClassNames(this, styles.main)}>
        <thead>
          <tr>
            {this.props.titles.map(title => <th key={title}>{title}</th>)}
            <th></th>
          </tr>
        </thead>
        <tbody>
          {this.props.children}
        </tbody>
      </table>
    )
  }
}

export default TableList
