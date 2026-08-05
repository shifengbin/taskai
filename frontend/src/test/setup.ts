import '@testing-library/jest-dom/vitest'

// jsdom does not implement PointerEvent / pointer-capture APIs that Radix UI
// (Toast, Dialog, Popover, Select ...) relies on in real browsers. Polyfill the
// HTMLElement surface so these components behave under jsdom.
if (!window.HTMLElement.prototype.hasPointerCapture) {
  window.HTMLElement.prototype.hasPointerCapture = () => false
}
if (!window.HTMLElement.prototype.setPointerCapture) {
  window.HTMLElement.prototype.setPointerCapture = () => {}
}
if (!window.HTMLElement.prototype.releasePointerCapture) {
  window.HTMLElement.prototype.releasePointerCapture = () => {}
}
if (!window.HTMLElement.prototype.scrollIntoView) {
  window.HTMLElement.prototype.scrollIntoView = () => {}
}

// jsdom does not simulate the native <select> dropdown: clicking an <option>
// in a real browser selects it and fires input/change, but jsdom leaves the
// select's value untouched. Polyfill that behavior so `user.click(option)`
// drives native <select> controls the same way the browser does.
document.addEventListener('click', (event) => {
  const target = event.target
  if (target instanceof HTMLOptionElement) {
    const select = target.closest('select')
    if (select && !select.multiple) {
      select.value = target.value
      select.dispatchEvent(new Event('input', {bubbles: true}))
      select.dispatchEvent(new Event('change', {bubbles: true}))
    }
  }
}, true)

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
