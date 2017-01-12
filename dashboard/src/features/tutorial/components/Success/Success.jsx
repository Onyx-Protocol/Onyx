import React from 'react'
import styles from './Success.scss'

class Success extends React.Component {

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
            {this.props.dismiss &&
              <div className={styles.skip}>
                <a onClick={this.props.handleDismiss}>{this.props.dismiss}</a>
              </div>
            }
          </div>
          <div className={styles.content}>
            <span className='glyphicon glyphicon-ok-sign'></span>
            <div className={styles.text}>
              {this.props.content.map(function (x, i){
                return <li key={i}>{x}</li>
              })}
            </div>

            {nextButton && nextButton}
          </div>
        </div>
    </div>
    )
  }
}

export default Success
