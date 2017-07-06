import React from 'react'
import { connect } from 'react-redux'
import styles from './Modal.scss'

class Modal extends React.Component {
  render() {
    let {
      dispatch,
      isShowing,
      body,
    } = this.props

    if (!isShowing) return null

    const close = () => dispatch({ type: 'HIDE_MODAL' })

    return(
      <div className={styles.main}>
        <div className={styles.backdrop} onClick={close}></div>
        <div className={`${this.props.options.wide && styles.wide} ${styles.modal}`}>
          {this.props.title && <div className={styles.title}>
            <span>{this.props.title}</span>

            <button type='button' className='close' onClick={close}>
              <span>&times;</span>
            </button>
          </div>}

          <div className={styles.content}>{body}</div>
        </div>
      </div>
    )
  }
}

const mapStateToProps = (state) => ({
  isShowing: state.app.modal.isShowing,
  title: state.app.modal.title,
  body: state.app.modal.body,
  options: state.app.modal.options,
})

// NOTE: ommitting a function for `mapDispatchToProps` passes `dispatch` as a
// prop to the component
export default connect(mapStateToProps)(Modal)
