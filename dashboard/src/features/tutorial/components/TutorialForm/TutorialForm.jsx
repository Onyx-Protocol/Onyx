import React from 'react'
import styles from './TutorialForm.scss'

class TutorialForm extends React.Component {

  render() {
    const userInput = this.props.userInput

    return (
      <div className={styles.container}>
        <div className={styles.tutorialContainer}>
          <div className={styles.header}>
            {this.props.title}
            <div className={styles.skip}>
              <a onClick={this.props.handleDismiss}>End tutorial</a>
            </div>
          </div>
          <div className={styles.list}>
            <div className={styles.listHeader}>
              {this.props.content['header']}
            </div>
            <table className={styles.listItemContainer}>
              <tbody>
              {this.props.content['steps'].map(function (contentLine, i){
                let str = contentLine['title']
                if (contentLine['type']) {
                  let replacement = userInput[contentLine['type']]
                  if ('index' in contentLine){
                    replacement = replacement[contentLine['index']]
                  }
                  str = contentLine['title'].replace('STRING', replacement['alias'])
                }
                let rows = [
                  <tr className={styles.listItem} key={i}>
                    <td className={styles.listBullet}>{i+1}</td>
                    <td>{str}</td>
                  </tr>
                ]
                if (contentLine['description']) {
                  rows.push (<tr className={styles.listItemDescription}>
                    <td></td>
                    <td key={i}>{contentLine['description']}</td>
                  </tr>)
                }
                return rows
              })}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    )
  }
}

export default TutorialForm
