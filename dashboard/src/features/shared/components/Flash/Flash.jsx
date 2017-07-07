import React from 'react'
import styles from './Flash.scss'

class Flash extends React.Component {
  componentWillReceiveProps(nextProps) {
    Object.keys(nextProps.messages).forEach(key => {
      const item = nextProps.messages[key]
      if (!item.displayed) {
        this.props.markFlashDisplayed(key)
      }
    })
  }

  render() {
    if (!this.props.messages || this.props.hideFlash) {
      return null
    }

    const messages = []
    // Flash messages are stored in an objecty key with a random UUID. If
    // multiple messages are displayed, we rely on the browser maintaining
    // object inerstion order of keys to display messages in the order they
    // were created.
    Object.keys(this.props.messages).forEach(key => {
      const item = this.props.messages[key]
      messages.push(
        <div className={`${styles.alert} ${styles[item.type]}`} key={key}>
          <div className={styles.content}>
            {item.title && <div><strong>{item.title}</strong></div>}
            {item.message}
          </div>

          <button type='button' className='close' onClick={() => this.props.dismissFlash(key)}>
            <span>&times;</span>
          </button>
        </div>)
    })

    return (
      <div className={styles.main}>
        {messages}
      </div>
    )
  }
}

import { connect } from 'react-redux'

const mapStateToProps = (state) => ({
  hideFlash: state.tutorial.isShowing && state.routing.locationBeforeTransitions.pathname.includes(state.tutorial.route)
})

export default connect(
  mapStateToProps
)(Flash)
