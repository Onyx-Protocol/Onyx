import React from 'react'
import styles from './TutorialModal.scss'
import { Link } from 'react-router'

class TutorialModal extends React.Component {

  render() {
    return (
      <div className={styles.main}>
        <div className={styles.backdrop} onClick={this.props.dismissTutorial}></div>
          <div className={styles.content}>
            <div className={styles.header}>
              {this.props.modalTitle}
            </div>
            <div className={styles.text}>
              {this.props.content.map(function (contentLine, i){
                if(contentLine['list']){
                  let list = []
                  contentLine['list'].forEach(function(listItem, j){
                    list.push(<li key={j}>{listItem}</li>)
                  })
                  return <ul className={styles.list}>{list}</ul>
                } else {
                  return <span className={styles.line} key={i}>{contentLine['line']}</span>
                }
              })}
            </div>
            <div className={styles.footer}>
              {this.props.modalNext && <Link to={this.props.route}>
                  <button key='showNext' className={`btn btn-primary ${styles.next}`} onClick={this.props.handleNext}>
                    {this.props.modalNext}
                  </button>
                </Link>}
              <button onClick={this.props.dismissTutorial} className={`btn btn-primary ${styles.dismiss}`}>{this.props.modalDismiss}</button>
            </div>
          </div>
      </div>
    )
  }
}

import { actions } from 'features/tutorial'
import { connect } from 'react-redux'

const mapStateToProps = (state) => ({
  tutorialRoute: state.tutorial.route,
  currentStep: state.tutorial.currentStep,
  showTutorial: state.routing.locationBeforeTransitions.pathname.startsWith(state.tutorial.route)
})

const mapDispatchToProps = ( dispatch ) => ({
  dismissTutorial: () => dispatch(actions.dismissTutorial)
})

export default connect(
  mapStateToProps,
  mapDispatchToProps
)(TutorialModal)
