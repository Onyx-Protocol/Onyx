import React from 'react'
import styles from './TutorialInfo.scss'
import { Link } from 'react-router'

class TutorialInfo extends React.Component {

  render() {
    let objectImage
    try {
      objectImage = require(`images/empty/${this.props.image}.svg`)
    } catch (err) { /* do nothing */ }

    const userInput = this.props.userInput
    const nextButton = <Link to={this.props.route} className={styles.nextWrapper}>
        <button key='showNext' className={`btn ${styles.next}`} onClick={this.props.handleNext}>
          Next: {this.props.button}
        </button>
      </Link>

    return (
      <div>
        <div className={styles.container}>
          {this.props.image && <img className={styles.image} src={objectImage} />}
          <div className={styles.text}>
            {this.props.content.map(function (contentLine, i){
              let str = contentLine
              if (contentLine['line']) { str = contentLine['line'] }
              if(contentLine['list']){
                let list = []
                contentLine['list'].forEach(function(listItem, j){
                  list.push(<tr key={j} className={styles.listItemGroup}>
                    <td className={styles.listBullet}>{j+1}</td>
                    <td>{listItem}</td>
                  </tr>)
                })
                return <table key={i} className={styles.listItemContainer}>
                  <tbody>{list}</tbody>
                </table>
              }
              if (contentLine['type']){
                let replacement = userInput[contentLine['type']]
                if ('index' in contentLine){
                  replacement = replacement[contentLine['index']]
                }
                str = str.replace('STRING', replacement['alias'])
              }

              return <p key={i}>{str}</p>
            })}
          </div>

          {nextButton}
        </div>
    </div>
    )
  }
}

export default TutorialInfo
