const keyNameMappings = {
  id: 'ID',
  type: 'Type',
  purpose: 'Purpose',
  isLocal: 'Local?',
  transactionId: 'Transaction ID',
  position: 'Position',
  spentOutput: 'Spent Output',
  assetId: 'Asset ID',
  assetAlias: 'Asset Alias',
  assetDefinition: 'Asset Definition',
  assetTags: 'Asset Tags',
  assetIsLocal: 'Asset Is Local?',
  amount: 'Amount',
  accountId: 'Account ID',
  accountAlias: 'Account Alias',
  accountTags: 'Account Tags',
  controlProgram: 'Control Program',
  issuanceProgram: 'Issuance Program',
  referenceData: 'Reference Data',
}

export default (inout) => {
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
