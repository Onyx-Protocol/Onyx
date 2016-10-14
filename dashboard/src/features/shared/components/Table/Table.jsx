import React from 'react'
import styles from './Table.scss'
import { Section } from 'features/shared/components'

class Table extends React.Component {
  isPlainObject(value) {
    return value != null && (typeof value == 'object') && !Array.isArray(value)
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
                {this.isPlainObject(item.value) ?
                  <td>
                    <pre className={styles.pre}>{JSON.stringify(item.value, null, '  ')}</pre>
                  </td> :
                  <td>{item.value}</td>
                }
              </tr>
            )}
          </tbody>
        </table>
      </Section>
    )
  }
}

export default Table
