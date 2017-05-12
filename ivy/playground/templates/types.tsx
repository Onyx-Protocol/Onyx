import { Template, ContractParameter } from 'ivy-compiler'

import { InputMap } from '../inputs/types'

export { Template }
export type ItemMap = { [s: string]: string }

export type ParameterType = {
  name: string,
  type: string
}

export type ValueInfo = {
  name: string,
  program?: string,
  asset?: string,
  amount?: string
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
  value: string,
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

