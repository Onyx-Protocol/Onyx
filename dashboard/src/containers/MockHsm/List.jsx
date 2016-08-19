import { connect } from 'react-redux'
import actions from '../../actions'
import ItemList from '../../components/ItemList'
import Item from '../../components/MockHsm/Item'

const type = "mockhsm"

const mapStateToProps = (state) => ({
  pages: state[type].pages,
  currentPage: state[type].currentPage,
  type: type,
  label: "key",
  listItemComponent: Item,
  keyProp: "xpub",
  skipQuery: true
})

const mapDispatchToProps = (dispatch) => ({
  getNextPage: () => dispatch(actions[type].displayNextPage()),
  getPrevPage: () => dispatch(actions[type].decrementPage()),
  showCreate: () => dispatch(actions[type].showCreate()),
})

export default connect(
  mapStateToProps,
  mapDispatchToProps
)(ItemList)
