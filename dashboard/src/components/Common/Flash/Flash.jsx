import React from 'react'

class Flash extends React.Component {
  componentWillReceiveProps(nextProps) {
    nextProps.messages.forEach((item, key) => {
      if (!item.displayed) {
        this.props.markFlashDisplayed(key)
      }
    })
  }

  render() {
    if (!this.props.messages) {
      return null
    }

    const messages = []
    this.props.messages.forEach((item, key) => {
      messages.push(
        <div className={`alert alert-${item.type}`} key={key}>
          {item.message}

          <button type='button' className='close' onClick={() => this.props.dismissFlash(key)}>
            <span>&times;</span>
          </button>
        </div>)
    })

    return (
      <div>
        {messages}
      </div>
    )
  }
}

export default Flash
