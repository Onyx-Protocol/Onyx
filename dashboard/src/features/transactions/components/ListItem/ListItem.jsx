import React from 'react'
import { Link } from 'react-router'
import { Summary } from 'features/transactions/components'
import { RelativeTime } from 'features/shared/components'
import styles from './ListItem.scss'

class ListItem extends React.Component {
  render() {
    const item = this.props.item
    const chevron = require('assets/images/chevron-green.png')

    return(
      <div className={styles.main}>
        <div className={styles.titleBar}>
          <div className={styles.title}>
            <label>Transaction ID:</label>
            <code>{item.id.slice(0,16) + '…'}</code>
            <span className={styles.timestamp}>
              <RelativeTime timestamp={item.timestamp} />
            </span>
          </div>
          <Link className={styles.viewLink} to={`/transactions/${item.id}`}>
            View Details
            <img src={chevron} className={styles.chevron} />
          </Link>
        </div>

        <Summary transaction={item} />
      </div>
    )
  }
}

export default ListItem
