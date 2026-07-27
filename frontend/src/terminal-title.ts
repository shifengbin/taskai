const ESCAPE = '\x1b'
const BEL = '\x07'
const C1_STRING_TERMINATOR = '\x9c'
const MAX_TERMINAL_TITLE_LENGTH = 4096

type TerminalTitleCommand = '' | '0' | '2'

export type TerminalTitleParserState =
  | {phase: 'text'}
  | {phase: 'escape'}
  | {phase: 'oscCommand'; command: TerminalTitleCommand}
  | {phase: 'oscCommandEscape'; command: TerminalTitleCommand}
  | {phase: 'oscTitle'; title: string}
  | {phase: 'oscTitleEscape'; title: string}
  | {phase: 'oscIgnored'}
  | {phase: 'oscIgnoredEscape'}
  | {phase: 'oscDiscard'}
  | {phase: 'oscDiscardEscape'}

export interface TerminalTitleParseResult {
  state: TerminalTitleParserState
  title?: string
}

interface TerminalTitleTransition {
  state: TerminalTitleParserState
  title?: string
}

export function createTerminalTitleParserState(): TerminalTitleParserState {
  return {phase: 'text'}
}

export function parseTerminalTitleOutput(state: TerminalTitleParserState, output: string): TerminalTitleParseResult {
  let nextState = state
  let title: string | undefined

  for (const character of output) {
    const transition = consumeCharacter(nextState, character)
    nextState = transition.state
    if (transition.title !== undefined) {
      title = transition.title
    }
  }

  return {state: nextState, title}
}

function consumeCharacter(state: TerminalTitleParserState, character: string): TerminalTitleTransition {
  switch (state.phase) {
    case 'text':
      return {state: character === ESCAPE ? {phase: 'escape'} : state}
    case 'escape':
      if (character === ']') {
        return {state: {phase: 'oscCommand', command: ''}}
      }
      return {state: character === ESCAPE ? state : {phase: 'text'}}
    case 'oscCommand':
      if (character === ';') {
        return {state: isTitleCommand(state.command) ? {phase: 'oscTitle', title: ''} : {phase: 'oscIgnored'}}
      }
      if (isOscTerminator(character)) {
        return {state: {phase: 'text'}}
      }
      if (character === ESCAPE) {
        return {state: {phase: 'oscCommandEscape', command: state.command}}
      }
      return {state: isTitleCommand(character) && state.command === '' ? {phase: 'oscCommand', command: character} : {phase: 'oscIgnored'}}
    case 'oscCommandEscape':
      if (character === '\\' || isOscTerminator(character)) {
        return {state: {phase: 'text'}}
      }
      if (character === ']') {
        return {state: {phase: 'oscCommand', command: ''}}
      }
      return {state: character === ESCAPE ? state : {phase: 'oscIgnored'}}
    case 'oscTitle':
      if (isOscTerminator(character)) {
        return completeTitle(state.title)
      }
      if (character === ESCAPE) {
        return {state: {phase: 'oscTitleEscape', title: state.title}}
      }
      return {state: appendTitle(state.title, character)}
    case 'oscTitleEscape':
      if (character === '\\' || isOscTerminator(character)) {
        return completeTitle(state.title)
      }
      if (character === ']') {
        return {state: {phase: 'oscCommand', command: ''}}
      }
      if (character === ESCAPE) {
        return {state}
      }
      return {state: appendTitle(state.title, ESCAPE + character)}
    case 'oscIgnored':
      if (isOscTerminator(character)) {
        return {state: {phase: 'text'}}
      }
      return {state: character === ESCAPE ? {phase: 'oscIgnoredEscape'} : state}
    case 'oscIgnoredEscape':
      if (character === '\\' || isOscTerminator(character)) {
        return {state: {phase: 'text'}}
      }
      if (character === ']') {
        return {state: {phase: 'oscCommand', command: ''}}
      }
      return {state: character === ESCAPE ? state : {phase: 'oscIgnored'}}
    case 'oscDiscard':
      if (isOscTerminator(character)) {
        return {state: {phase: 'text'}}
      }
      return {state: character === ESCAPE ? {phase: 'oscDiscardEscape'} : state}
    case 'oscDiscardEscape':
      if (character === '\\' || isOscTerminator(character)) {
        return {state: {phase: 'text'}}
      }
      if (character === ']') {
        return {state: {phase: 'oscCommand', command: ''}}
      }
      return {state: character === ESCAPE ? state : {phase: 'oscDiscard'}}
  }
}

function isTitleCommand(command: string): command is Exclude<TerminalTitleCommand, ''> {
  return command === '0' || command === '2'
}

function isOscTerminator(character: string): boolean {
  return character === BEL || character === C1_STRING_TERMINATOR
}

function appendTitle(title: string, characters: string): TerminalTitleParserState {
  if (title.length + characters.length > MAX_TERMINAL_TITLE_LENGTH) {
    return {phase: 'oscDiscard'}
  }
  return {phase: 'oscTitle', title: title + characters}
}

function completeTitle(title: string): TerminalTitleTransition {
  return {state: {phase: 'text'}, title}
}
