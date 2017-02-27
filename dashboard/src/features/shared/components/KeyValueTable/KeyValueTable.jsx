import React from 'react'
import styles from './KeyValueTable.scss'
import { Section } from 'features/shared/components'
import { Link } from 'react-router'

class KeyValueTable extends React.Component {
  shouldUsePre(item) {
    if (item.pre) return true

    return item.value != null && (typeof item.value == 'object')
  }

  stringify(value) {
    let separator = '  '

    if ((Array.isArray(value) && value.length <= 1) ||
         Object.keys(value).length <= 1) {
      for (let index in value) {
        if (typeof value[index] !== 'object') separator = null
      }
    }

    return JSON.stringify(value, null, separator)
  }

  renderValue(item) {
    let value = item.value
    if (this.shouldUsePre(item)) {
      value = <pre className={styles.pre}>{this.stringify(item.value)}</pre>
    }
    if (item.link) {
      value = <Link to={item.link}>{value}</Link>
    }

    if (value === undefined || value === null || value === '') {
      value = '-'
    }

    return value
  }

  render() {

    return(
      <Section
        title={this.props.title}
        actions={this.props.actions} >
        <table className={styles.table}>
          <tbody>
            {this.props.items.map((item) =>
              <tr key={`${item.label}`}>
                <td className={styles.label}>{item.label}</td>
                <td className={styles.value}>{this.renderValue(item)}</td>
              </tr>
            )}
          </tbody>
        </table>
      </Section>
    )
  }
}

export default KeyValueTable
