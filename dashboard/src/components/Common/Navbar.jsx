import React from 'react'

class Navbar extends React.Component {
  render() {
    return (
      <div className={`navbar navbar-default ${this.props.customStyles}`}>
        <div className='container'>
          {this.props.children}
        </div>
      </div>
    )
  }
}

export default Navbar
