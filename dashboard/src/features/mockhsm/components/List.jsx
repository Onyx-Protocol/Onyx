import { BaseList } from 'features/shared/components'
import ListItem from './ListItem'

const type = 'mockhsm'

const state = (state, ownProps) => ({
  ...BaseList.mapStateToProps(type, ListItem)(state, ownProps),
  skipQuery: true,
  label: 'Mock HSM Keys'
})

export default BaseList.connect(
  state,
  BaseList.mapDispatchToProps(type)
)
