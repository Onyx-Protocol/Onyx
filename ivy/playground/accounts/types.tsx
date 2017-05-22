export type Item = {
  id: string,
  alias: string
}

export type ItemMap = {
  [s: string]: Item
}

export type State = {
  itemMap: ItemMap,
  idList: string[],
  balanceMap: { [s: string]: number },
  shouldSeed: boolean
}
