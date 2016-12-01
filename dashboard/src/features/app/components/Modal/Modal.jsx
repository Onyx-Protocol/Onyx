import React from 'react'
import { connect } from 'react-redux'
import styles from './Modal.scss'
import actions from 'actions'

class Modal extends React.Component {
  render() {
    let {
      dispatch,
      isShowing,
      body,
      acceptAction,
      cancelAction
    } = this.props

    if (!isShowing) return null

    const accept = () => {
      dispatch(acceptAction)
      dispatch(actions.app.hideModal())
    }
    const cancel = cancelAction ? () => dispatch(cancelAction) : null
    const backdropAction = cancel || accept

    return(
      <div className={styles.main}>
        <div className={styles.backdrop} onClick={backdropAction}></div>
        <div className={`${this.props.options.wide && styles.wide} ${styles.content}`}>
          {body}

          <button className={`btn btn-${this.props.options.danger ? 'danger' : 'primary'} ${styles.accept}`} onClick={accept}>OK</button>
          {cancel && <button className={`btn btn-link ${styles.cancel}`} onClick={cancel}>Cancel</button>}
        </div>
      </div>
    )
  }
}

const mapStateToProps = (state) => ({
  isShowing: state.app.modal.isShowing,
  body: state.app.modal.body,
  acceptAction: state.app.modal.accept,
  cancelAction: state.app.modal.cancel,
  options: state.app.modal.options,
})

// NOTE: ommitting a function for `mapDispatchToProps` passes `dispatch` as a
// prop to the component
export default connect(mapStateToProps)(Modal)
