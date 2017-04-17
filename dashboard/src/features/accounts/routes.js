import AccountShow from 'features/accounts/components/AccountShow'
import List from 'features/accounts/components/List'
import New from 'features/accounts/components/New'
import { makeRoutes } from 'features/shared'

export default (store) => makeRoutes(store, 'account', List, New, AccountShow)
