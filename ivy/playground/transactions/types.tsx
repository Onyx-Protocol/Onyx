export type SpendFromAccount = {
  type: "spendFromAccount",
  accountId: string,
  assetId: string,
  amount: number
}

export type SpendUnspentOutput = {
  type: "spendUnspentOutput",
  outputId: string
}

export type ControlWithAccount = {
  type: "controlWithAccount",
  accountId: string,
  assetId: string,
  amount: number
}

export type Receiver = {
  controlProgram: string,
  expiresAt: string
}

export type ControlWithReceiver = {
  type: "controlWithReceiver",
  receiver: Receiver
  assetId: string,
  amount: number
}

export type Action = SpendFromAccount | ControlWithReceiver | ControlWithAccount | SpendUnspentOutput
