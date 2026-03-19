import { describe, expect, it } from 'vitest'

import { downloadRecordingUrl, formatBytes } from './recordings'

describe('downloadRecordingUrl', () => {
  it('encodes filename safely in URL path', () => {
    const url = downloadRecordingUrl('cam-1', 'front door clip #1.mp4')
    expect(url).toBe('/api/v1/cameras/cam-1/recordings/front%20door%20clip%20%231.mp4')
  })
})

describe('formatBytes', () => {
  it('formats bytes under 1 KB', () => {
    expect(formatBytes(999)).toBe('999 B')
  })

  it('formats KB and MB with one decimal place', () => {
    expect(formatBytes(1536)).toBe('1.5 KB')
    expect(formatBytes(5 * 1024 * 1024)).toBe('5.0 MB')
  })

  it('formats GB with two decimal places', () => {
    expect(formatBytes(3 * 1024 * 1024 * 1024)).toBe('3.00 GB')
  })
})
