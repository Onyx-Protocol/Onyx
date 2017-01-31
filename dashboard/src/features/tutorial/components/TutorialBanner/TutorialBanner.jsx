import React from 'react'
import { Link } from 'react-router'
import styles from './TutorialBanner.scss'

class TutorialBanner extends React.Component {

  render() {

    return (
      <div className={styles.header}>
        {this.props.title}
        {this.props.dismiss &&
          <div className={styles.skip}>
            {!this.props.showTutorial && <Link to={this.props.resumePath}>
              Resume tutorial
            </Link>}
            {this.props.showTutorial &&
            <a onClick={this.props.handleDismiss}>{this.props.dismiss}</a>}
          </div>}
      </div>
    )
  }
}

export default TutorialBanner
