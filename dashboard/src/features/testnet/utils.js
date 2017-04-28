const isBlockchainMismatch = (state) => {
  if (!state.core.onTestnet) {
    return false
  }

  return !!state.core.blockchainId && !!state.testnet.blockchainId &&
    state.core.blockchainId != state.testnet.blockchainId
}

const isCrosscoreRpcMismatch = (state) => {
  if (!state.core.onTestnet) {
    return false
  }

  return !!state.core.crosscoreRpcVersion && !!state.testnet.crosscoreRpcVersion &&
    state.core.crosscoreRpcVersion != state.testnet.crosscoreRpcVersion
}

export default {
  isBlockchainMismatch,
  isCrosscoreRpcMismatch,
}
