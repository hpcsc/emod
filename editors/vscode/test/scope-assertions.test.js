const assert = require('node:assert/strict')
const { spawnSync } = require('node:child_process')
const fs = require('node:fs')
const os = require('node:os')
const path = require('node:path')
const { describe, test } = require('node:test')

const extensionRoot = path.resolve(__dirname, '..')
const grammarPath = path.join(extensionRoot, 'syntaxes', 'emod.tmLanguage.json')
const harness = require.resolve('vscode-tmgrammar-test/dist/unit.js')

function assertionPath(name) {
  return path.join('test', 'scopes', name)
}

function readAssertions(name) {
  return fs.readFileSync(path.join(extensionRoot, assertionPath(name)), 'utf8')
}

function readGrammar() {
  return fs.readFileSync(grammarPath, 'utf8')
}

function escapeForRegExp(text) {
  return text.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

function runScopeTest(...args) {
  const run = spawnSync(process.execPath, [harness, ...args], {
    cwd: extensionRoot,
    encoding: 'utf8',
  })
  return { status: run.status, output: run.stdout + run.stderr }
}

function scratchDir(t) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'emod-scopes-'))
  t.after(() => fs.rmSync(dir, { recursive: true, force: true }))
  return dir
}

function writeScratchFile(t, name, contents) {
  const file = path.join(scratchDir(t), name)
  fs.writeFileSync(file, contents)
  return file
}

function grammarWithScopeRenamed({ from, to }) {
  const grammar = readGrammar()
  assert.ok(grammar.includes(`"${from}"`), `the grammar no longer names ${from}`)
  return grammar.replace(`"${from}"`, `"${to}"`)
}

function grammarWithFlatKeywordsExtended(words) {
  const grammar = JSON.parse(readGrammar())
  const alternation = grammar.repository.keywords.match
  const closingWordBoundary = /\)\\b$/
  assert.ok(
    closingWordBoundary.test(alternation),
    `the flat keyword rule is no longer a word alternation: ${alternation}`,
  )
  grammar.repository.keywords.match = alternation.replace(closingWordBoundary, `|${words.join('|')})\\b`)
  return JSON.stringify(grammar)
}

function assertionsWithScopeRenamed(name, { below, from, to }) {
  const lines = readAssertions(name).split('\n')
  const source = lines.findIndex((line) => line.trim() === below)
  assert.notEqual(source, -1, `${name} no longer contains the line "${below}"`)
  const expectation = lines.findIndex(
    (line, n) => n > source && new RegExp(`\\^+ ${escapeForRegExp(from)}$`).test(line),
  )
  assert.notEqual(expectation, -1, `no assertion below "${below}" requires ${from}`)
  lines[expectation] = lines[expectation].replace(from, to)
  return lines.join('\n')
}

function extensionConfigFor(t, grammar) {
  const dir = scratchDir(t)
  const config = path.join(dir, 'package.json')
  fs.writeFileSync(path.join(dir, 'emod.tmLanguage.json'), grammar)
  fs.writeFileSync(
    config,
    JSON.stringify({
      contributes: {
        languages: [{ id: 'emod', extensions: ['.emod'] }],
        grammars: [{ language: 'emod', scopeName: 'source.emod', path: './emod.tmLanguage.json' }],
      },
    }),
  )
  return config
}

function assertReportsMismatch({ status, output }, { file, sourceLine, required, prohibited, produced }) {
  assert.notEqual(status, 0, output)
  assert.match(output, new RegExp(`${escapeForRegExp(file)}:\\d+:\\d+:\\d+`))
  const named = [
    sourceLine,
    required && `missing required scopes: ${required}`,
    prohibited && `prohibited scopes: ${prohibited}`,
    `actual: source.emod ${produced}`,
  ].filter(Boolean)
  for (const fragment of named) {
    assert.ok(output.includes(fragment), `the failure report does not name "${fragment}":\n${output}`)
  }
}

describe('emod TextMate scope assertions', () => {
  test('the shipped grammar produces every scope the assertion files name', () => {
    const { status, output } = runScopeTest(assertionPath('*.emod'))

    assert.equal(status, 0, output)
  })

  test('a grammar that paints a token with another scope fails, naming the position and the scope produced', (t) => {
    const grammar = grammarWithScopeRenamed({ from: 'storage.type.emod', to: 'variable.other.emod' })
    const config = extensionConfigFor(t, grammar)
    const fieldAssertions = assertionPath('fields.emod')

    const run = runScopeTest('--config', config, fieldAssertions)

    assertReportsMismatch(run, {
      file: fieldAssertions,
      sourceLine: 'orderId string required',
      required: 'storage.type.emod',
      produced: 'variable.other.emod',
    })
  })

  for (const { keywords, sourceLine } of [
    { keywords: ['on', 'every'], sourceLine: 'on string required' },
    { keywords: ['type'], sourceLine: 'type string required' },
  ]) {
    test(`a grammar that colours ${keywords.join(' and ')} by word alone paints a field named after one`, (t) => {
      const config = extensionConfigFor(t, grammarWithFlatKeywordsExtended(keywords))
      const keywordFieldAssertions = assertionPath('unreserved-keywords.emod')

      const run = runScopeTest('--config', config, keywordFieldAssertions)

      assertReportsMismatch(run, {
        file: keywordFieldAssertions,
        sourceLine,
        prohibited: 'keyword.control.emod',
        produced: 'keyword.control.emod',
      })
    })
  }

  test('an assertion naming a scope the grammar does not produce fails, naming the position and the scope produced', (t) => {
    const assertions = assertionsWithScopeRenamed('declarations.emod', {
      below: 'command PlaceOrder {}',
      from: 'entity.name.function.emod',
      to: 'entity.name.tag.emod',
    })
    const mutated = writeScratchFile(t, 'declarations.emod', assertions)

    const run = runScopeTest(mutated)

    assertReportsMismatch(run, {
      file: 'declarations.emod',
      sourceLine: 'command PlaceOrder',
      required: 'entity.name.tag.emod',
      produced: 'entity.name.function.emod',
    })
  })
})
