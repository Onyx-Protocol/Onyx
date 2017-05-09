import { compileTemplate } from 'ivy-compiler'

import { CompilerError, Item, CompilerResult } from '../types'

export const mustCompileTemplate = (source: string): Item => {
  const res = compileTemplate(source)
  if (res.type == "compilerError") {
    throw res
  }
  return res
}

