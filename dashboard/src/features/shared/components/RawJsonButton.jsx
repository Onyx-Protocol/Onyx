import React from 'react'

class RawJsonButton extends React.Component {
  openJsonTab(item) {
    var rawJsonWindow = window.open(null, '_blank')
    rawJsonWindow.document.write(`
      <html>
        <head><title>${this.props.title}</title></head>
        <body>
          <pre style="word-wrap: break-word; white-space: pre-wrap;">
${JSON.stringify(item, null, 2)}
          </pre>
        </body>
      </html>`)
  }

  render() {
    return (
      <a className='btn btn-link' onClick={this.openJsonTab.bind(this, this.props.item)}>
        Raw JSON
      </a>
    )
  }
}

export default RawJsonButton
