import { Template, CompilerError as _CompilerError } from 'ivy-compiler'

export type CompilerError = _CompilerError
export type Item = Template
export type ItemMap = { [s: string]: Item }
export type State = {
  itemMap: ItemMap,
  idList: string[],
  source: string,
  selected: string
}

