import React from 'react'
import {
  NumberField,
  XpubField,
} from "../Common"

class KeyConfiguration extends React.Component {
  constructor(props) {
    super(props)

    this.state = { keys: '' }
  }

  render() {
    const keyCountChange = event => {
      let count = Math.min(event.target.value, 10)
      let existing = this.state.keys || 0

      if (count > existing) {
        for (let i = 0; i < count - existing; i++) {
          this.props.xpubs.addField()
        }
      } else if (count < existing) {
        for (let i = 0; i < existing - count; i++) {
          this.props.xpubs.removeField()
        }
      }

      this.setState({ keys: count })
    }

    return(
      <div>
        <NumberField title="Keys" fieldProps={{value: this.state.keys, onChange: keyCountChange}} />
        <NumberField title="Quorum" hint="Number of keys required for transfer" fieldProps={this.props.quorum} />

        {this.props.xpubs.map((xpub, index) =>
          <XpubField
            key={index}
            index={index}
            mockhsmKeys={this.props.mockhsmKeys}
            fieldProps={xpub}
          />)}
      </div>
    )
  }


}

export default KeyConfiguration
