import { CompilerResult, CompiledTemplate } from '../types'

// Creates an empty CompiledTemplate to use upon compiler errors.
export const makeEmptyTemplate = (source: string, error: string): CompiledTemplate => {
  return {
    name: '',
    params: [],
    clauses: [],
    value: '',
    bodyBytecode: '',
    bodyOpcodes: '',
    recursive: false,
    source,
    error
  }
}

// Converts undefined, array attributes into empty arrays.
export const formatCompilerResult = (result: CompilerResult): CompilerResult => {
  if (result.contracts.length < 1) {
    throw '0 contracts returned from compiler'
  }

  return ({
    ...result,
    contracts: result.contracts.map(orig => {
      const contract = {
        ...orig,
        params: orig.params || [],
        clauses: orig.clauses || []
      } as CompiledTemplate

      const clauses = contract.clauses.map(clause => ({
        ...clause,
        params: clause.params || [],
        reqs: clause.reqs || [],
        mintimes: clause.mintimes || [],
        maxtimes: clause.maxtimes || [],
        values: clause.values || [],
        hashCalls: clause.hashCalls || []
      }))

      return ({
        ...contract,
        clauses
      } as CompiledTemplate)
    })
  } as CompilerResult)
}

// Returns the last contract in the list returned from the compiler.
// By convention, this is the default contract.
export const getDefaultContract = (source: string, result: CompilerResult): CompiledTemplate => {
  if (result.contracts.length < 1) {
    throw '0 contracts returned from compiler'
  }

  return ({
    ...result.contracts[result.contracts.length-1],
    source,
    error: ''
  } as CompiledTemplate)
}
