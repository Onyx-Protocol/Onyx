import { Template as OldTemplate, ContractParameter } from 'ivy-compiler'

import { InputMap } from '../inputs/types'

export type SourceMap = { [s: string]: string }

export type Param = {
  name: string,
  type: string
}

export type HashCall = {
  hashType: string,
  arg: string,
  argType: string
}

export type ValueInfo = {
  name: string,
  program?: string,
  asset?: string,
  amount?: string
}

export type ClauseInfo = {
  name: string,
  args: Param[],
  mintimes: string[],
  maxtimes: string[],
  hashCalls: HashCall[],
  valueInfo: ValueInfo[]
}

export type CompiledTemplate = {
  name: string,
  source: string,
  program: string,
  opcodes: string,
  error: string,
  params: Param[],
  value: string,
  clauseInfo: ClauseInfo[]
}

export type TemplateState = {
  sourceMap: SourceMap,
  idList: string[],
  source: string,
  inputMap?: InputMap,
  compiled?: CompiledTemplate,
}

export { OldTemplate }
