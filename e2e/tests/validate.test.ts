import { describe, it, expect } from 'vitest'
import { launchTerminal } from 'tuistory'
import { getExecutablePath } from '../testUtils'

describe('emod validate', () => {
  it('valid file with all four patterns exits cleanly', async () => {
    const executablePath = getExecutablePath()

    const session = await launchTerminal({
      command: 'sh',
      args: ['-c', `${executablePath} validate internal/parser/testdata/all_patterns.emod; echo "EXIT:$?"`],
      cols: 120,
      rows: 24,
    })

    try {
      const text = await session.waitForText('EXIT:', { timeout: 10000 })
      expect(text).toContain('EXIT:0')
    } finally {
      session.close()
    }
  })

  it('invalid file prints diagnostics to stderr', async () => {
    const executablePath = getExecutablePath()

    const session = await launchTerminal({
      command: 'sh',
      args: ['-c', `${executablePath} validate internal/parser/testdata/invalid.emod 2>&1; echo "EXIT:$?"`],
      cols: 120,
      rows: 24,
    })

    try {
      const text = await session.waitForText('EXIT:', { timeout: 10000 })
      expect(text).toContain('invalid.emod')
      expect(text).toMatch(/\d+:/)
      expect(text).toContain('foobar')
      expect(text).toContain('EXIT:1')
    } finally {
      session.close()
    }
  })

  it('nonexistent file prints error', async () => {
    const executablePath = getExecutablePath()

    const session = await launchTerminal({
      command: 'sh',
      args: ['-c', `${executablePath} validate /tmp/nonexistent.emod 2>&1; echo "EXIT:$?"`],
      cols: 120,
      rows: 24,
    })

    try {
      const text = await session.waitForText('EXIT:', { timeout: 10000 })
      expect(text).toContain('nonexistent.emod')
      expect(text).toContain('EXIT:1')
    } finally {
      session.close()
    }
  })

  it('no file argument prints error', async () => {
    const executablePath = getExecutablePath()

    const session = await launchTerminal({
      command: 'sh',
      args: ['-c', `${executablePath} validate 2>&1; echo "EXIT:$?"`],
      cols: 120,
      rows: 24,
    })

    try {
      const text = await session.waitForText('EXIT:', { timeout: 10000 })
      expect(text).toContain('validate requires exactly one file argument')
      expect(text).toContain('EXIT:1')
    } finally {
      session.close()
    }
  })
})
