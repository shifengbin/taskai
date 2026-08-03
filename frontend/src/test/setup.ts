import '@testing-library/jest-dom/vitest'

class ResizeObserver {
  private static readonly callbacks = new Set<ResizeObserverCallback>()
  private readonly callback: ResizeObserverCallback

  constructor(callback: ResizeObserverCallback) {
    this.callback = callback
    ResizeObserver.callbacks.add(callback)
  }

  observe() {}
  unobserve() {}
  disconnect() {
    ResizeObserver.callbacks.delete(this.callback)
  }

  static notify() {
    for (const callback of ResizeObserver.callbacks) {
      callback([], {} as ResizeObserver)
    }
  }

  static reset() {
    ResizeObserver.callbacks.clear()
  }
}

Object.defineProperty(window, 'ResizeObserver', {value: ResizeObserver})
