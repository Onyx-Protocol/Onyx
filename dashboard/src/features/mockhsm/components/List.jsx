import { BaseList, TableList } from 'features/shared/components'
import ListItem from './ListItem'

const type = 'mockhsm'

export default BaseList.connect(
  BaseList.mapStateToProps(type, ListItem, {
    skipQuery: true,
    label: 'Mock HSM Keys',
    wrapperComponent: TableList,
    wrapperProps: {
      titles: ['Alias', 'Xpub']
    }
  }),
  BaseList.mapDispatchToProps(type)
)
