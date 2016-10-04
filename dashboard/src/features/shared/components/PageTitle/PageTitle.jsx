import React from 'react'
import styles from './PageTitle.scss'

class PageTitle extends React.Component {
  render() {
    const breadcrumbs = this.props.breadcrumbs || []

    return(
      <div className={styles.main}>
        <div className={styles.navigation}>
          {breadcrumbs.map((item, index) =>
            <span key={`breadcrumb-${index}`}>{item}</span>
          )}
          <span className='title'>{this.props.title}</span>
        </div>

        {this.props.actions && <div>{this.props.actions}</div>}
      </div>
    )
  }
}

export default PageTitle
