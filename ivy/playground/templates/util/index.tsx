import { compileTemplate } from 'ivy-compiler'

import { CompilerError, Item } from '../types'

export const mustCompileTemplate = (source: string): Item => {
  const res = compileTemplate(source)
  if (res.type == "compilerError") {
    throw res
  }
  return res
}


export const isError = (template: Item|CompilerError): template is CompilerError => {
  return template.type === "compilerError"
}
