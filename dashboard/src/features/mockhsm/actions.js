import { baseListActions, baseCreateActions } from 'features/shared/actions'
import { chainClient } from 'utility/environment'

const type = 'mockhsm'
const clientApi = () => chainClient().mockHsm.keys

export default {
  ...baseCreateActions(type, {
    className: 'MockHsm',
    clientApi,
  }),
  ...baseListActions(type, {
    className: 'MockHsm',
    clientApi,
  }),
}
