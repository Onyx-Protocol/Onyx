import React from 'react'
import { SelectField, XpubField } from 'features/shared/components'

const rangeOptions = [1,2,3,4,5,6].map(val => ({label: val, value: val}))

class KeyConfiguration extends React.Component {
  constructor(props) {
    super(props)

    this.state = { keys: 1 }
    this.props.xpubs.addField()
  }

  render() {
    const {
      quorum,
      quorumHint,
      xpubs
    } = this.props

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
      let maxKeys = parseInt(event.target.value) || 0
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

    const quorumOptions = rangeOptions.slice(0, this.state.keys)

    return(
      <div>
        <SelectField options={rangeOptions}
          title='Keys'
          skipEmpty={true}
          fieldProps={{
            value: this.state.keys,
            onChange: keyCountChange,
          }} />

        <SelectField options={quorumOptions}
          title='Quorum'
          skipEmpty={true}
          hint={quorumHint}
          fieldProps={{
            ...quorum,
            onChange: quorumChange,
          }} />

        {xpubs.map((xpub, index) =>
          <XpubField
            key={`xpub-${index}`}
            index={index}
            typeProps={xpub.type}
            valueProps={xpub.value}
          />)}
      </div>
    )
  }
}

export default KeyConfiguration
