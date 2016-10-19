import React from 'react'
import styles from './KeyValueTable.scss'
import { Section } from 'features/shared/components'

class KeyValueTable extends React.Component {
  renderPre(value) {
    return value != null && (typeof value == 'object')
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
                {this.renderPre(item.value) ?
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

export default KeyValueTable
