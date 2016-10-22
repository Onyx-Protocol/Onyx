import React from 'react'
import { KeyValueTable } from 'features/shared/components'

class ListItem extends React.Component {
  render() {
    const item = {...this.props.item}

    const after = item.after.split('-')[0].split(':')
    const blockHeight = after[0]
    const blockPosition = after[1]

    // int max (2147483647) is used to indicate that a feed
    // hasn't yet been read from.
    const hasStarted = blockPosition != '2147483647'

    const options = [
      {label: 'ID', value: item.id}
    ]

    if (item.alias) options.push({label: 'Alias', value: item.alias})
    options.push({label: 'Filter', value: item.filter, link: `/transactions?filter=${item.filter}`, pre: true})

    if (hasStarted) {
      options.push({label: 'Last Acknowledged', value: {blockHeight, blockPosition}})
    } else {
      options.push({label: 'Last Acknowledged', value: 'None'})
    }

    return(
      <KeyValueTable items={options} />
    )
  }
}

export default ListItem
