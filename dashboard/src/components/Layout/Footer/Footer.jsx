import React from 'react'
import styles from './Footer.scss'

class Footer extends React.Component {
  render() {
    return (
      <footer className={`${styles.footer}`}>
        <div className="container">
          <div className="row">
            <div className="col-sm-6 text-left">
              Build: <code>{this.props.buildCommit}</code> on {this.props.buildDate}
            </div>
            <div className="col-sm-6 text-right">
              © Chain
            </div>
          </div>
        </div>
      </footer>
    )
  }
}

export default Footer
