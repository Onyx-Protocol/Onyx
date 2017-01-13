import React from 'react'
import styles from './Form.scss'

class Form extends React.Component {

  render() {
    return (
      <div className={styles.container}>
        <div className={styles.header}>
          {this.props.title}
        </div>
        <div className={styles.content}>
          <div className={styles.listHeader}>
            {this.props.content['header']}
          </div>
          <table className={styles.listItemContainer}>
            {this.props.content['steps'].map(function (x, i){
              return <div className={styles.listItem}>
                <tr key={i}>
                  <td className={styles.listBullet}>{i+1}</td>
                  <td>{x['title']}</td>
                </tr>
                { x['description'] && <tr className={styles.listItemDescription}>
                  <td></td>
                  <td>{x['description']}</td>
                </tr> }
              </div>
            })}
          </table>
        </div>
      </div>
    )
  }
}

export default Form
