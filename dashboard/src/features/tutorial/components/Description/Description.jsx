import React from 'react'
import styles from './Description.scss'

class Description extends React.Component {

  render() {
    const nextButton = <div className={styles.next}>
      <button key='showNext' className='btn btn-primary' onClick={this.props.handleNext}>
        {this.props.button}
      </button>
    </div>

    return (
      <div>
        <div className={styles.container}>
          <div className={styles.header}>
            {this.props.title}
            <div className={styles.skip}>
              <a onClick={this.props.handleDismiss}>{this.props.dismiss}</a>
            </div>
          </div>
          <div className={styles.content}>
            <div className={styles.text}>{this.props.content}</div>

            {nextButton && nextButton}
          </div>
        </div>
    </div>
    )
  }
}

export default Description
