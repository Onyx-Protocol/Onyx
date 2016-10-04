import React from 'react'
import {
  BaseShow,
  PageTitle,
  Table,
  Container,
} from 'features/shared/components'
import { Summary } from './'

class Show extends BaseShow {
  actionDetails(action) {
    let details = [{label: 'Action', value: action.action}]
    if (action.purpose) details.push({label: 'Action Purpose', value: action.purpose})

    details = details.concat([
      {label: 'Asset ID', value: action.asset_id},
      {label: 'Asset Alias', value: action.asset_alias},
      {label: 'Asset Tags', value: action.asset_tags},
      {label: 'Amount', value: action.amount},
    ])

    if (action.account_id) {
      details = details.concat([
        {label: 'Account ID', value: action.account_id},
        {label: 'Account Alias', value: action.account_alias},
        {label: 'Account Tags', value: action.account_tags},
      ])
    }

    if (action.control_program) details.push({label: 'Control Program', value: action.control_program})
    if (action.issuance_program) details.push({label: 'Issuance Program', value: action.issuance_program})

    details.push({label: 'Reference Data', value: action.reference_data})
    return details
  }

  render() {
    const item = this.props.item

    let view
    if (item) {
      const title = <span>
        {'Transaction '}
        <code>{item.id}</code>
      </span>

      view = <div>
        <PageTitle title={title} />

        <Container title='Summary'>
          <Summary transaction={item} />
        </Container>

        <Table
          title='Details'
          items={[
            {label: 'ID', value: item.id},
            {label: 'Timestamp', value: item.timestamp},
            {label: 'Block ID', value: item.block_id},
            {label: 'Position', value: item.position},
            {label: 'Reference Data', value: item.reference_data},
          ]}
        />

        {item.inputs.map((input, index) =>
          <Table
            key={index}
            title={index == 0 ? 'Inputs' : ''}
            items={this.actionDetails(input)}
          />
        )}

        {item.outputs.map((output, index) =>
          <Table
            key={index}
            title={index == 0 ? 'Outputs' : ''}
            items={this.actionDetails(output)}
          />
        )}

      </div>
    }
    return this.renderIfFound(view)
  }
}

// Container

import { actions } from 'features/transactions'
import { connect } from 'react-redux'

const mapStateToProps = (state, ownProps) => ({
  item: state.transaction.items[ownProps.params.id]
})

const mapDispatchToProps = ( dispatch ) => ({
  fetchItem: (id) => dispatch(actions.fetchItems({filter: `id='${id}'`}))
})

export default connect(
  mapStateToProps,
  mapDispatchToProps
)(Show)
