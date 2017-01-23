const keyNameMappings = {
  type: 'Type',
  purpose: 'Purpose',
  is_local: 'Local?',
  transaction_id: 'Transaction ID',
  position: 'Position',
  output_id: 'Output ID',
  spent_output: 'Spent Output',
  asset_id: 'Asset ID',
  asset_alias: 'Asset Alias',
  asset_definition: 'Asset Definition',
  asset_tags: 'Asset Tags',
  asset_is_local: 'Asset Is Local?',
  amount: 'Amount',
  account_id: 'Account ID',
  account_alias: 'Account Alias',
  account_tags: 'Account Tags',
  control_program: 'Control Program',
  issuance_program: 'Issuance Program',
  reference_data: 'Reference Data',
}

export const buildInOutDisplay = (inout) => {
  const copy = {...inout}
  const details = []

  Object.keys(keyNameMappings).forEach(key => {
    if (copy[key] != null) {
      details.push({label: keyNameMappings[key], value: copy[key]})
      delete copy[key]
    }
  })

  Object.keys(copy).forEach(key => {
    if (copy[key] != null) {
      details.push({label: key, value: copy[key]})
    }
  })

  return details
}
