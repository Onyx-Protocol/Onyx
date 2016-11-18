import { BaseList } from 'features/shared/components'
import ListItem from './ListItem'

const type = 'unspent'

const newStateToProps = (state, ownProps) => ({
  ...BaseList.mapStateToProps(type, ListItem)(state, ownProps),
  skipCreate: true,
  label: 'Unspent outputs'
})

export default BaseList.connect(
  newStateToProps,
  BaseList.mapDispatchToProps(type)
)
