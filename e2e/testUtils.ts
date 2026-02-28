import { resolve } from 'node:path'

export function getExecutablePath(): string {
  const executablePath = process.env.EXECUTABLE
  if (!executablePath) {
    throw new Error(
      'EXECUTABLE environment variable is required. Set it to the path of your CLI executable.',
    )
  }
  return resolve(executablePath)
}
