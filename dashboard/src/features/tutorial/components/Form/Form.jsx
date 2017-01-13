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
          <div className={styles.listItemContainer}>
            {this.props.content['steps'].map(function (x, i){
              let listItem = <div>
                <li className={styles.listItem} key={i}>
                  <div className={styles.listBullet}>{i+1}</div> {x['title']}
                </li>
                { x['description'] && <li className={styles.listItemDescription}>
                  {x['description']}
                </li> }
              </div>
              return listItem
            })}
          </div>
        </div>
      </div>
    )
  }
}

export default Form
