import {describe, expect, it} from 'vitest'

import {clampTaskTreeWidth} from './types'

describe('clampTaskTreeWidth', () => {
  it('将拖拽产生的小数宽度归一化为整数', () => {
    expect(clampTaskTreeWidth(311.25390625, 1280)).toBe(311)
  })
})
