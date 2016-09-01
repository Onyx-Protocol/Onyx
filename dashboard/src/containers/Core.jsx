import { connect } from 'react-redux'
import Index from '../components/Core/Index'

const mapStateToProps = (state) => ({
  core: state.core
})

const mapDispatchToProps = (dispatch) => ({})

export default connect(
  mapStateToProps,
  mapDispatchToProps
)(Index)
