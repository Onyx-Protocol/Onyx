import React from 'react'
import { KeyValueTable } from 'features/shared/components'

class ListItem extends React.Component {
  render() {
    const item = {...this.props.item}
    delete item.id

    const sumBy = []
    Object.keys(item.sum_by).forEach(key =>
      sumBy.push({label: key, value: item.sum_by[key]})
    )

    return(
      <KeyValueTable items={[
        {label: 'Amount', value: item.amount},
        ...sumBy
      ]} />
    )
  }
}

export default ListItem
