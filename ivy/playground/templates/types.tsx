import { Template, CompilerError as _CompilerError } from 'ivy-compiler'

export type CompilerError = _CompilerError
export type Item = Template
export type ItemMap = { [s: string]: Item }

export type ParameterType = {
  name: string,
  type: string
}

export type ValueInfo = {
  name: string,
  program?: string,
  assetAmount?: string
}

export type ClauseInfo = {
  name: string,
  args: ParameterType[],
  valueInfo: ValueInfo[]
}

export type CompilerResult = {
  name: string,
  source: string,
  program: string,
  opcodes: string,
  error: string,
  params: ParameterType[],
  clauseInfo: ClauseInfo[]
}

export type State = {
  itemMap: ItemMap,
  idList: string[],
  source: string,
  selected: string,
  compiled?: CompilerResult
}

