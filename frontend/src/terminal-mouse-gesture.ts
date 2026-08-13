export interface TerminalSelectionPosition {
  clientX: number
  clientY: number
  text: string
}

export type TerminalSelectionCompleteHandler = (position: TerminalSelectionPosition) => void
export type TerminalContextMenuHandler = () => void

export interface TerminalSelectionRange {
  start: {x: number, y: number}
  end: {x: number, y: number}
}

interface PreservedSelection extends TerminalSelectionRange {
  text: string
}

const dragThreshold = 4
const multiClickDelay = 300

interface MouseEventSnapshot {
  altKey: boolean
  button: number
  buttons: number
  clientX: number
  clientY: number
  ctrlKey: boolean
  detail: number
  metaKey: boolean
  screenX: number
  screenY: number
  shiftKey: boolean
}

interface PointerSequence {
  down: MouseEventSnapshot
  dragging: boolean
}

interface PendingClick {
  down: MouseEventSnapshot
  up: MouseEventSnapshot
}

export class TerminalMouseGesture {
  private readonly replayedEvents = new WeakSet<Event>()
  private pointer?: PointerSequence
  private pendingClick?: PendingClick
  private clickTimer?: number
  private selectionTimer?: number
  private selectionComplete?: TerminalSelectionCompleteHandler
  private contextMenu?: TerminalContextMenuHandler
  private preservedSelection?: PreservedSelection
  private restoringSelection = false
  private disposed = false

  constructor(
    private readonly element: HTMLElement,
    private readonly mouseTrackingActive: () => boolean,
    private readonly selectionText: () => string,
    selectionComplete?: TerminalSelectionCompleteHandler,
    contextMenu?: TerminalContextMenuHandler,
    private readonly selectionRange?: () => TerminalSelectionRange | undefined,
    private readonly restoreSelection?: (selection: PreservedSelection) => boolean,
  ) {
    this.selectionComplete = selectionComplete
    this.contextMenu = contextMenu
    this.element.addEventListener('mousedown', this.handleMouseDown, true)
    this.element.addEventListener('mousemove', this.handleHoverMouseMove, true)
    this.element.addEventListener('mouseup', this.handleElementMouseUp, true)
    this.element.addEventListener('contextmenu', this.handleContextMenu, true)
    this.element.addEventListener('keydown', this.clearPreservedSelection, true)
    this.element.addEventListener('paste', this.clearPreservedSelection, true)
    this.element.addEventListener('wheel', this.clearPreservedSelection, true)
  }

  setSelectionCompleteHandler(handler?: TerminalSelectionCompleteHandler): void {
    this.selectionComplete = handler
  }

  setContextMenuHandler(handler?: TerminalContextMenuHandler): void {
    this.contextMenu = handler
  }

  dispose(): void {
    if (this.disposed) {
      return
    }
    this.disposed = true
    this.cancelPending()
    this.element.removeEventListener('mousedown', this.handleMouseDown, true)
    this.element.removeEventListener('mousemove', this.handleHoverMouseMove, true)
    this.element.removeEventListener('mouseup', this.handleElementMouseUp, true)
    this.element.removeEventListener('contextmenu', this.handleContextMenu, true)
    this.element.removeEventListener('keydown', this.clearPreservedSelection, true)
    this.element.removeEventListener('paste', this.clearPreservedSelection, true)
    this.element.removeEventListener('wheel', this.clearPreservedSelection, true)
  }

  restorePreservedSelection(): boolean {
    if (!this.preservedSelection || this.restoringSelection || this.disposed) {
      return false
    }
    this.restoringSelection = true
    let restored = false
    try {
      restored = this.restoreSelection?.(this.preservedSelection) === true
    } catch {
      restored = false
    } finally {
      this.restoringSelection = false
    }
    if (!restored) {
      this.preservedSelection = undefined
    }
    return restored
  }

  private readonly handleMouseDown = (event: MouseEvent): void => {
    if (this.disposed || this.replayedEvents.has(event)) {
      return
    }
    this.preservedSelection = undefined
    if (event.button === 2) {
      stopMouseEvent(event)
      return
    }
    if (event.button !== 0) {
      return
    }
    this.clearClickTimer()
    this.pointer = {down: snapshotMouseEvent(event), dragging: false}
    this.element.ownerDocument.addEventListener('mousemove', this.handleMouseMove, true)
    this.element.ownerDocument.addEventListener('mouseup', this.handleMouseUp, true)
    stopMouseEvent(event)
  }

  private readonly handleElementMouseUp = (event: MouseEvent): void => {
    if (!this.disposed && !this.replayedEvents.has(event) && event.button === 2) {
      stopMouseEvent(event)
    }
  }

  private readonly handleHoverMouseMove = (event: MouseEvent): void => {
    if (this.disposed || this.replayedEvents.has(event) || event.buttons || !this.mouseTrackingActive()) {
      return
    }
    const range = this.selectionRange?.()
    const text = this.selectionText()
    if (!range || !text) {
      return
    }
    this.preservedSelection = {
      start: {...range.start},
      end: {...range.end},
      text,
    }
  }

  private readonly clearPreservedSelection = (): void => {
    this.preservedSelection = undefined
  }

  private readonly handleContextMenu = (event: MouseEvent): void => {
    if (this.disposed) {
      return
    }
    this.preservedSelection = undefined
    stopMouseEvent(event)
    this.contextMenu?.()
  }

  private readonly handleMouseMove = (event: MouseEvent): void => {
    if (this.disposed || this.replayedEvents.has(event) || !this.pointer) {
      return
    }
    if (!this.pointer.dragging && exceedsDragThreshold(this.pointer.down, event)) {
      this.pendingClick = undefined
      this.clearClickTimer()
      this.pointer.dragging = true
      this.replayMouseEvent('mousedown', this.pointer.down, this.mouseTrackingActive())
    }
    if (this.pointer.dragging) {
      this.replayDocumentMouseEvent('mousemove', snapshotMouseEvent(event), this.mouseTrackingActive())
    }
    stopMouseEvent(event)
  }

  private readonly handleMouseUp = (event: MouseEvent): void => {
    if (this.disposed || this.replayedEvents.has(event) || !this.pointer || event.button !== 0) {
      return
    }
    const pointer = this.pointer
    this.pointer = undefined
    this.removePointerListeners()
    if (pointer.dragging) {
      this.replayDocumentMouseEvent('mouseup', snapshotMouseEvent(event), this.mouseTrackingActive())
      stopMouseEvent(event)
      this.publishSelectionComplete(event.clientX, event.clientY)
      return
    }

    stopMouseEvent(event)
    this.pendingClick = {down: pointer.down, up: snapshotMouseEvent(event)}
    const clickCount = normalizedClickCount(pointer.down.detail)
    if (clickCount === 3) {
      this.completePendingClick()
      return
    }
    this.clickTimer = window.setTimeout(() => this.completePendingClick(), multiClickDelay)
  }

  private completePendingClick(): void {
    this.clearClickTimer()
    const click = this.pendingClick
    this.pendingClick = undefined
    if (!click || this.disposed) {
      return
    }
    const clickCount = normalizedClickCount(click.down.detail)
    const forceSelection = clickCount > 1 && this.mouseTrackingActive()
    this.replayMouseEvent('mousedown', click.down, forceSelection, clickCount)
    this.replayMouseEvent('mouseup', click.up, forceSelection, clickCount)
    if (forceSelection) {
      this.publishSelectionComplete(click.up.clientX, click.up.clientY)
    } else if (clickCount > 1) {
      this.publishSelectionComplete(click.up.clientX, click.up.clientY)
    }
  }

  private replayMouseEvent(type: 'mousedown' | 'mouseup', source: MouseEventSnapshot, forceSelection: boolean, detail = source.detail): void {
    const event = this.createMouseEvent(type, source, forceSelection, detail)
    this.replayedEvents.add(event)
    this.element.dispatchEvent(event)
  }

  private replayDocumentMouseEvent(type: 'mousemove' | 'mouseup', source: MouseEventSnapshot, forceSelection: boolean): void {
    const event = this.createMouseEvent(type, source, forceSelection, source.detail)
    this.replayedEvents.add(event)
    this.element.dispatchEvent(event)
  }

  private createMouseEvent(type: 'mousedown' | 'mousemove' | 'mouseup', source: MouseEventSnapshot, forceSelection: boolean, detail: number): MouseEvent {
    const isMac = typeof navigator !== 'undefined' && /Macintosh|Mac OS|iPhone|iPad/.test(navigator.userAgent)
    return new MouseEvent(type, {
      altKey: forceSelection && isMac ? true : source.altKey,
      bubbles: true,
      button: source.button,
      buttons: type === 'mouseup' ? 0 : source.buttons || 1,
      cancelable: true,
      clientX: source.clientX,
      clientY: source.clientY,
      ctrlKey: source.ctrlKey,
      detail,
      metaKey: source.metaKey,
      screenX: source.screenX,
      screenY: source.screenY,
      shiftKey: forceSelection && !isMac ? true : source.shiftKey,
    })
  }

  private cancelPending(): void {
    this.pointer = undefined
    this.pendingClick = undefined
    this.clearClickTimer()
    this.clearSelectionTimer()
    this.removePointerListeners()
  }

  private clearClickTimer(): void {
    if (this.clickTimer !== undefined) {
      window.clearTimeout(this.clickTimer)
      this.clickTimer = undefined
    }
  }

  private publishSelectionComplete(clientX: number, clientY: number): void {
    this.clearSelectionTimer()
    this.selectionTimer = window.setTimeout(() => {
      this.selectionTimer = undefined
      this.selectionComplete?.({clientX, clientY, text: this.selectionText()})
    }, 0)
  }

  private clearSelectionTimer(): void {
    if (this.selectionTimer !== undefined) {
      window.clearTimeout(this.selectionTimer)
      this.selectionTimer = undefined
    }
  }

  private removePointerListeners(): void {
    this.element.ownerDocument.removeEventListener('mousemove', this.handleMouseMove, true)
    this.element.ownerDocument.removeEventListener('mouseup', this.handleMouseUp, true)
  }
}

function snapshotMouseEvent(event: MouseEvent): MouseEventSnapshot {
  return {
    altKey: event.altKey,
    button: event.button,
    buttons: event.buttons,
    clientX: event.clientX,
    clientY: event.clientY,
    ctrlKey: event.ctrlKey,
    detail: event.detail,
    metaKey: event.metaKey,
    screenX: event.screenX,
    screenY: event.screenY,
    shiftKey: event.shiftKey,
  }
}

function exceedsDragThreshold(start: MouseEventSnapshot, event: MouseEvent): boolean {
  const horizontalDistance = event.clientX - start.clientX
  const verticalDistance = event.clientY - start.clientY
  return horizontalDistance * horizontalDistance + verticalDistance * verticalDistance > dragThreshold * dragThreshold
}

function normalizedClickCount(detail: number): 1 | 2 | 3 {
  if (detail >= 3) {
    return 3
  }
  return detail === 2 ? 2 : 1
}

function stopMouseEvent(event: MouseEvent): void {
  event.preventDefault()
  event.stopImmediatePropagation()
}
