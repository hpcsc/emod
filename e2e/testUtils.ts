import { spawn } from 'node:child_process'
import { mkdtempSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join, resolve } from 'node:path'
import { onTestFinished } from 'vitest'

export function getExecutablePath(): string {
  const executablePath = process.env.EXECUTABLE
  if (!executablePath) {
    throw new Error(
      'EXECUTABLE environment variable is required. Set it to the path of your CLI executable.',
    )
  }
  return resolve(executablePath)
}

export function buildTag(): string {
  const tag = process.env.BUILD_TAG
  if (!tag) {
    throw new Error('BUILD_TAG environment variable is required. Set it to the tag the emod binary was built with.')
  }
  return tag
}

export function scratchDir(): string {
  const dir = mkdtempSync(join(tmpdir(), 'emod-e2e-'))
  onTestFinished(() => rmSync(dir, { recursive: true, force: true }))
  return dir
}

export interface Result {
  stdout: string
  stderr: string
  status: number | null
}

// runEmod does not block, so a fake server in this process can still answer
// the requests emod makes.
export function runEmod(
  cwd: string,
  args: string[],
  env: Record<string, string> = {},
  executable = getExecutablePath(),
): Promise<Result> {
  return new Promise((done, fail) => {
    const child = spawn(executable, args, { cwd, env: { ...process.env, ...env } })
    let stdout = ''
    let stderr = ''
    child.stdout.on('data', (chunk) => (stdout += chunk))
    child.stderr.on('data', (chunk) => (stderr += chunk))
    child.on('error', fail)
    child.on('close', (status) => done({ stdout, stderr, status }))
  })
}
