import { Template, ContractParameter, CompilerError as _CompilerError } from 'ivy-compiler'

import { InputMap } from '../inputs/types'

export type CompilerError = _CompilerError
export { Template }
export type ItemMap = { [s: string]: string }

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

export type TemplateState = {
  itemMap: ItemMap,
  idList: string[],
  source: string,
  inputMap?: InputMap,
  compiled?: CompilerResult,
  contractParameters?: ContractParameter[]
}

