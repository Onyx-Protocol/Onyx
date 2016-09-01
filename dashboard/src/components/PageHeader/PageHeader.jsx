import React from 'react'
import styles from "./PageHeader.scss"

class PageHeader extends React.Component {
  render() {
    let button = null
    if (this.props.showActionButton) {
      button = <button
        className={`btn btn-default btn-md ${styles.button}`}
        onClick={this.props.buttonAction}>
          <span className='glyphicon glyphicon-plus'></span>
          &nbsp;
          {this.props.buttonLabel}
      </button>
    }

    return (
      <div className={styles.main + " " + (this.props.additionalStyles || "")}>
        <h1 className="page-header">{this.props.title}</h1>
        {button}
      </div>
    )
  }
}

export default PageHeader
