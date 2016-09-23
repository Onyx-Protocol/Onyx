import { mapStateToProps, mapDispatchToProps, connect } from '../Base/List'
import Item from '../../components/Balance/Item'

const type = 'balance'

const newStateToProps = (state) => {
  const props =  {
    ...mapStateToProps(type, Item)(state),
    skipCreate: true,
  }
  props.searchState.sumBy = state[type].listView.sumBy
  return props
}

export default connect(
  newStateToProps,
  mapDispatchToProps(type)
)
