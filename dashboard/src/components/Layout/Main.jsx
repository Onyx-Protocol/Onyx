import React from 'react'
import { Flash } from '../Common'
import Navigation from './Navigation/Main'
import Footer from './Footer/Footer'
import styles from './Main.scss'

class Main extends React.Component {
  constructor(props) {
    super(props)

    this.toggleDropdown = this.toggleDropdown.bind(this)
  }

  toggleDropdown(event) {
    event.stopPropagation()
    this.props.toggleDropdown()
  }

  render() {
    return (
      <div
        className={styles.main}
        onClick={this.props.closeDropdown} >
        <Navigation
          dropdownState={this.props.dropdownState}
          toggleDropdown={this.toggleDropdown} />

        <div className='container'>
          <Flash messages={this.props.flashMessages}
            markFlashDisplayed={this.props.markFlashDisplayed}
            dismissFlash={this.props.dismissFlash}
          />

          {this.props.children}
        </div>

        <Footer
          buildCommit={this.props.buildCommit}
          buildDate={this.props.buildDate} />
      </div>
    )
  }
}

export default Main
