import { Location } from './ast'

export class ExtendableError extends Error {
  constructor(public message: string, public name: string, public location?: Location) {
    super();
    this.stack = (new Error()).stack;
  }
}

export class NameError extends ExtendableError {
  constructor(message: string, location?: Location) {
    super(message, "NameError", location)
  }
}

export class BugError extends ExtendableError {
  constructor(message: string) {
    super(message, "BugError")
  }
}

export class IvyTypeError extends ExtendableError {
  constructor(message: string, location?: Location) {
    super(message, "IvyTypeError", location)
  }
}

export class ValueError extends ExtendableError {
  constructor(message: string, location?: Location) {
    super(message, "ValueError", location)
  }
}