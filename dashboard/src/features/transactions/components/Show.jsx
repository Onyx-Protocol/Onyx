import React from 'react'
import {
  BaseShow,
  PageTitle,
  PageContent,
  Table,
  Section,
} from 'features/shared/components'
import { Summary } from './'

class Show extends BaseShow {
  inoutDetails(inout) {
    let details = [{label: 'Type', value: inout.type}]
    if (inout.purpose) details.push({label: 'Action Purpose', value: inout.purpose})

    details = details.concat([
      {label: 'Asset ID', value: inout.asset_id},
      {label: 'Asset Alias', value: inout.asset_alias},
      {label: 'Asset Tags', value: inout.asset_tags},
      {label: 'Amount', value: inout.amount},
    ])

    if (inout.account_id) {
      details = details.concat([
        {label: 'Account ID', value: inout.account_id},
        {label: 'Account Alias', value: inout.account_alias},
        {label: 'Account Tags', value: inout.account_tags},
      ])
    }

    if (inout.control_program) details.push({label: 'Control Program', value: inout.control_program})
    if (inout.issuance_program) details.push({label: 'Issuance Program', value: inout.issuance_program})

    details.push({label: 'Reference Data', value: inout.reference_data})
    return details
  }

  render() {
    const item = this.props.item

    let view
    if (item) {
      const title = <span>
        {'Transaction '}
        <code>{item.id.substring(0,8) + '…'}</code>
      </span>

      view = <div>
        <PageTitle title={title} />

        <PageContent>
          <Section title='Summary'>
            <Summary transaction={item} />
          </Section>

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
              items={this.inoutDetails(input)}
            />
          )}

          {item.outputs.map((output, index) =>
            <Table
              key={index}
              title={index == 0 ? 'Outputs' : ''}
              items={this.inoutDetails(output)}
            />
          )}
        </PageContent>
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
