export type DataWitness = {
  type: "data",
  value: string
}

export type KeyId = {
  xpub: string,
  derivationPath: string[]
}

export type SignatureWitness = {
  type: "signature",
  quorum: 1,
  keys: KeyId[],
  signatures: string[]
}

export type WitnessComponent = DataWitness | SignatureWitness

export type SigningInstruction = {
  position: number,
  witnessComponents: WitnessComponent[]
}

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
