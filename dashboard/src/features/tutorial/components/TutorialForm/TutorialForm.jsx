import React from 'react'
import styles from './TutorialForm.scss'

class TutorialForm extends React.Component {

  render() {
    const userInput = this.props.userInput

    return (
      <div className={styles.container}>
        <div className={styles.tutorialContainer}>
          <div className={styles.header}>
            Tutorial: {this.props.content['header']}
          </div>
          <div className={styles.list}>
            <table className={styles.listItemContainer}>
              {this.props.content['steps'].map(function (contentLine, i){
                let title = contentLine['title']
                if (contentLine['type']) {
                  let replacement = userInput[contentLine['type']]
                  if ('index' in contentLine){
                    replacement = replacement[contentLine['index']]
                  }
                  title = contentLine['title'].replace('STRING', replacement['alias'])
                }
                let rows = [
                  <tr key={i}>
                    <td className={styles.listBullet}>{i+1}</td>
                    <td>{title}</td>
                  </tr>
                ]
                if (contentLine['description']) {
                  let descriptionResult = ''
                  contentLine['description'].forEach(function (descriptionLine, j){
                    let description = descriptionLine['line']
                    if (descriptionLine['type']) {
                      let replacement = userInput[descriptionLine['type']]
                      if ('index' in descriptionLine){
                        replacement = replacement[descriptionLine['index']]
                      }
                      description = descriptionLine['line'].replace('STRING', replacement['alias'])
                    }
                    descriptionResult += description
                  })

                  rows.push (<tr className={styles.listItemDescription}>
                    <td></td>
                    <td key={i}>{descriptionResult}</td>
                  </tr>)
                }

                return <tbody className={styles.listItemGroup}>{rows}</tbody>
              })}
            </table>
          </div>
        </div>
      </div>
    )
  }
}

export default TutorialForm
