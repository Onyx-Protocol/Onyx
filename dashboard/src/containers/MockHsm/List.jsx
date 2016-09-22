import { mapStateToProps, mapDispatchToProps, connect } from '../Base/List'
import Item from '../../components/MockHsm/Item'

const type = 'mockhsm'

const state = (state) => ({
  ...mapStateToProps(type, Item)(state),
  skipQuery: true,
  label: 'Mock HSM Keys'
})

const dispatch = (dispatch) => ({
  ...mapDispatchToProps(type)(dispatch),
  updateQuery: null
})


export default connect(
  state,
  dispatch
)
