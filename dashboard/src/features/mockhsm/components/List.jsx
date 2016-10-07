import { BaseList } from 'features/shared/components'
import ListItem from './ListItem'

const type = 'mockhsm'

const state = (state, ownProps) => ({
  ...BaseList.mapStateToProps(type, ListItem)(state, ownProps),
  skipQuery: true,
  label: 'Mock HSM Keys'
})

const dispatch = (dispatch) => ({
  ...BaseList.mapDispatchToProps(type)(dispatch),
  updateQuery: null
})

export default BaseList.connect(
  state,
  dispatch
)
