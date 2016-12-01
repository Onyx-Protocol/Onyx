import React from 'react'
import {
  BaseShow,
  PageTitle,
  PageContent,
  KeyValueTable,
  Section,
  RawJsonButton,
} from 'features/shared/components'

import { Summary } from './'
import { buildInOutDisplay } from 'features/transactions/utility'

class Show extends BaseShow {


  render() {
    const item = this.props.item

    let view
    if (item) {
      const title = <span>
        {'Transaction '}
        &nbsp;<code>{item.id}</code>
      </span>

      view = <div>
        <PageTitle title={title} />

        <PageContent>
          <Section
            title='Summary'
            actions={[
              <RawJsonButton key='raw-json' item={item} />
            ]}>
            <Summary transaction={item} />
          </Section>

          <KeyValueTable
            title='Details'
            items={[
              {label: 'ID', value: item.id},
              {label: 'Timestamp', value: item.timestamp},
              {label: 'Block ID', value: item.block_id},
              {label: 'Block Height', value: item.block_height},
              {label: 'Position', value: item.position},
              {label: 'Local?', value: item.is_local},
              {label: 'Reference Data', value: item.reference_data},
            ]}
          />

          {item.inputs.map((input, index) =>
            <KeyValueTable
              key={index}
              title={index == 0 ? 'Inputs' : ''}
              items={buildInOutDisplay(input)}
            />
          )}

          {item.outputs.map((output, index) =>
            <KeyValueTable
              key={index}
              title={index == 0 ? 'Outputs' : ''}
              items={buildInOutDisplay(output)}
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
