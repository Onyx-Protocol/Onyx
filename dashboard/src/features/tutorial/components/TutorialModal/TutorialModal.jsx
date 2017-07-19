import React from 'react'
import styles from './TutorialModal.scss'
import { Link } from 'react-router'
import componentClassNames from 'utility/componentClassNames'

class TutorialModal extends React.Component {

  render() {
    return (
      <div className={componentClassNames(this, styles.main)}>
        <div className={styles.backdrop} onClick={this.props.dismissTutorial}></div>
          <div className={styles.content}>
            <div className={styles.header}>
              {this.props.title}
            </div>
            <div className={styles.text}>
              {this.props.content.map(function (contentLine, i){
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
                } else {
                  let value = contentLine
                  if (typeof(value) === 'object') { value = value['line'] }
                  return <p key={i}>{value}</p>
                }
              })}
            </div>
            <div className={styles.footer}>
              <button onClick={this.props.dismissTutorial} className={`btn btn-primary ${styles.dismiss}`}>{this.props.dismiss}</button>
              {this.props.button && <Link to={this.props.route}>
                  <button key='showNext' className={`btn btn-primary ${styles.next}`} onClick={this.props.handleNext}>
                    {this.props.button}
                  </button>
                </Link>}
            </div>
          </div>
      </div>
    )
  }
}

import { connect } from 'react-redux'

const mapStateToProps = (state) => ({
  tutorialRoute: state.tutorial.route,
  currentStep: state.tutorial.currentStep,
  showTutorial: state.routing.locationBeforeTransitions.pathname.includes(state.tutorial.route)
})

const mapDispatchToProps = ( dispatch ) => ({
  dismissTutorial: () => dispatch({ type: 'DISMISS_TUTORIAL' })
})

export default connect(
  mapStateToProps,
  mapDispatchToProps
)(TutorialModal)
