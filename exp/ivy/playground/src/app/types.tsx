import * as accounts from '../accounts/types'
import * as assets from '../assets/types'
import * as contracts from '../contracts/types'
import * as templates from '../templates/types'

export type AppState = {
  accounts: accounts.State,
  assets: assets.State,
  contracts: contracts.ContractsState
  templates: templates.TemplateState,
  routing: any
}
