import React from 'react'
import { Link } from 'react-router'
import { Summary } from 'features/transactions/components'
import { RelativeTime } from 'features/shared/components'
import styles from './ListItem.scss'

class ListItem extends React.Component {
  render() {
    const item = this.props.item

    const allInouts = item.inputs.concat(item.outputs)

    const confidential = allInouts.some(io => io.confidential == 'yes')
    const readable = allInouts.some(io => io.readable == 'yes')

    const classNames = [styles.main]
    if (!readable) {
      classNames.push(styles.notReadable)
    }

    return(
      <div className={classNames.join(' ')}>
        <div className={styles.titleBar}>
          <div className={styles.title}>
            <label>Transaction ID:</label>
            &nbsp;<code>{item.id}</code>&nbsp;

            <span className={styles.timestamp}>
              <RelativeTime timestamp={item.timestamp} />
            </span>
          </div>
          <span className={styles.icons}>
            {confidential && <span className='glyphicon glyphicon-lock' />}
            {readable && <span className={`glyphicon glyphicon-eye-open ${styles.iconReadable}`} />}
            {!readable && <span className={`glyphicon glyphicon-eye-close ${styles.iconNotReadable}`} />}
          </span>
          <Link className={styles.viewLink} to={`/transactions/${item.id}`}>
            View details
          </Link>
        </div>

        {readable && <Summary transaction={item} />}
      </div>
    )
  }
}

export default ListItem
