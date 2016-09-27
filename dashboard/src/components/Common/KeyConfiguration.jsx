import React from 'react'
import {
  NumberField,
  XpubField,
} from '../Common'

class KeyConfiguration extends React.Component {
  constructor(props) {
    super(props)

    this.state = { keys: '' }
  }

  render() {
    // Override onChange here rather than in a redux-form normalizer because
    // we're using component state (keys) to determine the max value
    const quorumChange = (event, maxKeys) => {
      let quorum = parseInt(typeof(event) == 'object' ? event.target.value : event)
      if (isNaN(quorum)) { return }

      if (maxKeys == undefined) {
        maxKeys = parseInt(this.state.keys || 0)
      }

      if (quorum > maxKeys) { quorum = maxKeys }

      this.props.quorum.onChange(quorum)
    }

    const keyCountChange = event => {
      let maxKeys = Math.min(event.target.value, 10)
      let existing = this.state.keys || 0

      if (maxKeys > existing) {
        for (let i = 0; i < maxKeys - existing; i++) {
          this.props.xpubs.addField()
        }
      } else if (maxKeys < existing) {
        for (let i = 0; i < existing - maxKeys; i++) {
          this.props.xpubs.removeField()
        }
      }

      this.setState({ keys: maxKeys })
      quorumChange(this.props.quorum.value, maxKeys)
    }

    return(
      <div>
        <NumberField title='Keys' fieldProps={{
          value: this.state.keys,
          onChange: keyCountChange,
          min: 0,
          max: 10
        }} />
        <NumberField title='Quorum' hint='Number of keys required for transfer' fieldProps={{
          ...this.props.quorum,
          onChange: quorumChange,
          min: 0,
          max: 10
        }} />

        {this.props.xpubs.map((xpub, index) =>
          <XpubField
            key={`xpub-${index}`}
            index={index}
            fieldProps={xpub}
          />)}
      </div>
    )
  }


}

export default KeyConfiguration
