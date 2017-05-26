import { ClauseParameterType, ContractParameterType, HashType, InputMap } from '../inputs/types'

export type SourceMap = { [s: string]: string }

export type Param = {
  name: string
  declaredType: ContractParameterType | ClauseParameterType
  inferredType?: HashType
}

export type HashCall = {
  hashType: string,
  arg: string,
  argType: string
}

export type ClauseReq = {
  name: string
  asset: string
  amount: string
}

export type Value = {
  name: string,
  program?: string,
  asset?: string,
  amount?: string
}

export type Clause = {
  name: string
  params: Param[]
  reqs: ClauseReq[]
  mintimes: string[]
  maxtimes: string[]
  hashCalls: HashCall[]
  values: Value[]
}

export type CompiledTemplate = {
  name: string
  params: Param[]
  clauses: Clause[]
  value: string
  bodyBytecode: string
  bodyOpcodes: string
  recursive: boolean
  source: string
  error?
}

export type CompilerResult = {
  contracts: CompiledTemplate[]
  programMap: { [s: string]: string }
  error: string
}

export type TemplateState = {
  sourceMap: SourceMap
  idList: string[]
  protectedIdList: string[]
  source: string
  sourceChanged: boolean
  inputMap?: InputMap
  compiled?: CompiledTemplate
  showLockInputErrors: boolean
  error?
}
