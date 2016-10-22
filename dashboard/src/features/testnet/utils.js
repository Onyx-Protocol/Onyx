const isBlockchainMismatch = (state) => {
  if (!state.core.onTestnet) {
    return false
  }

  return !!state.core.blockchainId && !!state.testnet.blockchainId &&
    state.core.blockchainId != state.testnet.blockchainId
}

const isNetworkMismatch = (state) => {
  if (!state.core.onTestnet) {
    return false
  }

  return !!state.core.networkRpcVersion && !!state.testnet.rpcVersion &&
    state.core.networkRpcVersion != state.testnet.rpcVersion
}

export default {
  isBlockchainMismatch,
  isNetworkMismatch,
}
