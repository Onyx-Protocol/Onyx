import React from 'react'
import styles from './Form.scss'

class Form extends React.Component {

  render() {
    const userInput = this.props.userInput

    return (
      <div className={styles.container}>
        <div className={styles.header}>
          {this.props.title}
          <div className={styles.skip}>
            <a onClick={this.props.handleDismiss}>End tutorial</a>
          </div>
        </div>
        <div className={styles.content}>
          <div className={styles.listHeader}>
            {this.props.content['header']}
          </div>
          <table className={styles.listItemContainer}>
            <tbody>
            {this.props.content['steps'].map(function (x, i){
              let str = x['title']
              
              if (x['type']) {
                if (x['type'] == 'account'){
                  str = x['title'].replace('STRING', userInput['accounts'][x['index']]['alias'])
                } else {
                  str = x['title'].replace('STRING', userInput[x['type']]['alias'])
                }
              }
              let rows = [
                <tr className={styles.listItem} key={i}>
                  <td className={styles.listBullet}>{i+1}</td>
                  <td>{str}</td>
                </tr>
              ]
              if (x['description']) {
                rows.push (<tr className={styles.listItemDescription}>
                  <td></td>
                  <td key={i}>{x['description']}</td>
                </tr>)
              }
              return rows
            })}
            </tbody>
          </table>
        </div>
      </div>
    )
  }
}

export default Form
