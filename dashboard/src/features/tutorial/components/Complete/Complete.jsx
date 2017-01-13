import React from 'react'
import styles from './Complete.scss'


class Complete extends React.Component {

  render() {
    return (
      <div className={styles.main}>
        <div className={styles.backdrop}></div>
          <div className={styles.content}>
            <div className={styles.header}>
              5-minute Tutorial completed!
            </div>
            <div className={styles.text}>
              <p>
                In this tutorial, you learned how to:<br />
              </p>
              <p>
                1. create and issue assets<br />
                2. create accounts<br />
                3. Build, sign and submit transactions<br />
              </p>
              <p>
                  If you need to revisit this tutorial, you can click Tutorial in
                  the sidenav dropdown menu. For detailed information on how Chain
                  Core works, please take a look at the <a href='/docs' target='_blank'>
                    Developer Documentation
                  </a>.
              </p>
            </div>
            <button onClick={this.props.handleDismiss} className={`btn btn-primary ${styles.tutorialButton}`}>{this.props.dismiss}</button>
          </div>
      </div>
    )
  }
}

export default Complete
