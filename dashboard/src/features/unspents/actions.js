import { baseListActions } from 'features/shared/actions'
import { chainClient } from 'utility/environment'

export default baseListActions('unspent', {
  clientApi: () => chainClient().unspentOutputs
})
