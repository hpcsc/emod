import { describe, expect, it } from 'vitest'
import { buildTag, runEmod, scratchDir } from '../testUtils'

describe('emod version', () => {
  it('prints the tag the binary was built from', async () => {
    const result = await runEmod(scratchDir(), ['version'])

    expect(result.status).toBe(0)
    expect(result.stdout.trim()).toBe(buildTag())
  })

  it('--version prints the same tag', async () => {
    const result = await runEmod(scratchDir(), ['--version'])

    expect(result.status).toBe(0)
    expect(result.stdout).toContain(buildTag())
  })
})
