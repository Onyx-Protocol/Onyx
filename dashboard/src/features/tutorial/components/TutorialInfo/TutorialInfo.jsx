import React from 'react'
import styles from './TutorialInfo.scss'
import { Link } from 'react-router'

class TutorialInfo extends React.Component {

  render() {
    const userInput = this.props.userInput
    const nextButton = <div className={styles.next}>
      <Link to={this.props.route}>
        <button key='showNext' className='btn btn-primary' onClick={this.props.handleNext}>
          {this.props.button}
        </button>
      </Link>
    </div>

    return (
      <div>
        <div className={styles.container}>
          <div className={styles.content}>
            {this.props.logo && <span className={`glyphicon ${this.props.logo}`}></span>}
            <div className={styles.text}>
              {this.props.content.map(function (contentLine, i){
                let str = contentLine['line']
                if (contentLine['type']){
                  let replacement = userInput[contentLine['type']]
                  if ('index' in contentLine){
                    replacement = replacement[contentLine['index']]
                  }
                  str = str.replace('STRING', replacement['alias'])
                }

                return <li key={i}>{str}</li>
              })}
            </div>

            {nextButton && nextButton}
          </div>
        </div>
    </div>
    )
  }
}

export default TutorialInfo
