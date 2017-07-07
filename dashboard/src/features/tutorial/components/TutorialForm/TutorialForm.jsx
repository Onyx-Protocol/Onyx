import React from 'react'
import styles from './TutorialForm.scss'

class TutorialForm extends React.Component {
  constructor() {
    super()

    this.state = { showFixed: false }

    // We must bind in the constructor so that we have a reference to the bound
    // function to remove from the window event listener later.
    this.handleScroll = this.handleScroll.bind(this)
  }

  handleScroll(event) {
    const scrollTop = event.srcElement.scrollingElement.scrollTop

    // Hardcoding visual distance between top of screen and top of TutorialForm
    // component to create smooth scrolling effect.
    this.setState({showFixed: scrollTop > 140})
  }

  componentDidMount() {
    window.addEventListener('scroll', this.handleScroll)
  }

  componentWillUnmount() {
    window.removeEventListener('scroll', this.handleScroll)
  }

  render() {
    const userInput = this.props.userInput

    return (
      <div className={styles.container}>
        <div className={`${styles.tutorialContainer} ${this.state.showFixed && styles.fixedTutorial}`}>
          <div className={styles.header}>
            {this.props.content['header']}
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
                let bullet = <td className={styles.listBullet}>{i+1}</td>
                let description = ''

                if (contentLine['description']) {
                  let descriptionResult = []
                  contentLine['description'].forEach( (descriptionLine, j) => {
                    let description = descriptionLine
                    if (description['line']) { description = description['line'] }

                    if (descriptionLine['type']) {
                      let replacement = userInput[descriptionLine['type']] || descriptionLine['type']
                      if ('index' in descriptionLine){
                        replacement = replacement[descriptionLine['index']]
                      }

                      if (replacement.hasOwnProperty('alias')) {
                        replacement = replacement['alias'] || ''
                      }

                      description.split('STRING').forEach( (item, k, arr) => {
                        descriptionResult.push(item)
                        let replacementText = k < arr.length - 1 && <span key={`item-input-${j}-${k}`} className={styles.userInputData}>"{replacement}"</span>
                        descriptionResult.push(replacementText)
                      })
                    } else {
                      descriptionResult.push(description)
                    }
                  })

                  description = <span className={styles.listItemDescription}>{descriptionResult}</span>
                }

                return <tr key={`item-${i}`}>
                  {bullet}
                  <td className={styles.listItemContent}>
                    <span className={styles.listItemTitle}>{title}</span>
                    {description}
                  </td>
                </tr>
              })}
            </table>
          </div>
        </div>
      </div>
    )
  }
}

export default TutorialForm
