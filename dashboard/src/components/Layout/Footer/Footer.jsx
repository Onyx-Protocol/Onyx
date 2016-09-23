import React from 'react'
import styles from './Footer.scss'

class Footer extends React.Component {
  render() {
    let buildString = <span>
      Build: <code>{this.props.buildCommit}</code> on {this.props.buildDate}
    </span>
    if (this.props.buildCommit === 'Local development') {
      buildString = <span>Build: <code>{this.props.buildCommit}</code></span>
    }

    return (
      <footer className={`${styles.footer}`}>
        <div className='container'>
          <div className='row'>
            <div className='col-sm-6 text-left'>
              {buildString}
            </div>
            <div className='col-sm-6 text-right'>
              © Chain
            </div>
          </div>
        </div>
      </footer>
    )
  }
}

export default Footer
