import { describe, it, expect, beforeEach } from 'vitest';
import { SourcePanel } from '../static/source-panel.js';
import { createStore } from '../static/store.js';
import { bus } from '../static/bus.js';

function setupSourcePanel(text) {
  document.body.innerHTML = `
    <div id="source-gutter"></div>
    <pre id="source-colours"></pre>
    <textarea id="source-input"></textarea>
    <div id="caret-line-diagnostics"></div>
  `;
  const store = createStore();
  store.dom.sourceGutter = document.getElementById('source-gutter');
  store.dom.sourceColours = document.getElementById('source-colours');
  store.dom.sourceInput = document.getElementById('source-input');
  store.dom.caretLineDiagnostics = document.getElementById('caret-line-diagnostics');
  store.dom.sourceInput.value = text || '';
  SourcePanel.initSourcePanel(store);
  return store;
}

function report(store, diagnostics) {
  store.diagnostics = diagnostics;
  SourcePanel.showDiagnostics(store);
}

function gutterEntries() {
  return Array.from(document.querySelectorAll('#source-gutter [title]')).map((entry) => ({
    title: entry.title,
    severity: entry.dataset.severity,
  }));
}

beforeEach(() => {
  document.body.innerHTML = '';
});

// A browser changes the text it is typed into without running the value setter
// that a script calls, and announces the change with an input event.
function typeIntoPanel(panel, text) {
  Object.getOwnPropertyDescriptor(HTMLTextAreaElement.prototype, 'value').set.call(panel, text);
  panel.dispatchEvent(new Event('input', { bubbles: true }));
}

describe('SourcePanel', () => {
  describe('source colours', () => {
    const coloursText = () => document.getElementById('source-colours').textContent;

    it('draw the text typed into the panel', () => {
      const store = setupSourcePanel('emod = 1\n');

      typeIntoPanel(store.dom.sourceInput, 'emod = 1\n\nmodel "Orders" {\n}\n');

      expect(coloursText()).toBe('emod = 1\n\nmodel "Orders" {\n}\n');
    });

    const painted = () => Array.from(document.getElementById('source-colours').childNodes)
      .map((node) => [node.textContent, node.dataset ? node.dataset.category : null]);

    it('paint the tokens of an answer for the text the panel holds', () => {
      const store = setupSourcePanel('emod = 1\n');

      SourcePanel.showTokens(store, 'emod = 1\n', [
        { start: 0, end: 4, category: 'variable.member' },
        { start: 7, end: 8, category: 'number' },
      ]);

      expect(painted()).toEqual([['emod', 'variable.member'], [' = ', null], ['1', 'number'], ['\n', null]]);
    });

    describe('while the answer for the newest text is still to come', () => {
      const source = 'emod = 1\n# a note\nx = 22\n';
      const answered = (store) => SourcePanel.showTokens(store, source, [
        { start: 0, end: 4, category: 'variable.member' },
        { start: 7, end: 8, category: 'number' },
        { start: 9, end: 17, category: 'comment' },
        { start: 18, end: 19, category: 'variable.member' },
        { start: 22, end: 24, category: 'number' },
      ]);

      it('keep the colour of each token outside the edited stretch on its own characters', () => {
        const store = setupSourcePanel(source);
        answered(store);

        typeIntoPanel(store.dom.sourceInput, 'emod = 1\ny = 3\n# a note\nx = 22\n');

        expect(painted()).toEqual([
          ['emod', 'variable.member'], [' = ', null], ['1', 'number'], ['\ny = 3\n', null],
          ['# a note', 'comment'], ['\n', null], ['x', 'variable.member'], [' = ', null], ['22', 'number'], ['\n', null],
        ]);
      });

      it('move the tokens after a deletion back onto their characters', () => {
        const store = setupSourcePanel(source);
        answered(store);

        typeIntoPanel(store.dom.sourceInput, 'emod = 1\nx = 22\n');

        expect(painted()).toEqual([
          ['emod', 'variable.member'], [' = ', null], ['1', 'number'], ['\n', null],
          ['x', 'variable.member'], [' = ', null], ['22', 'number'], ['\n', null],
        ]);
      });

      it("give text typed inside a token that token's colour", () => {
        const store = setupSourcePanel(source);
        answered(store);

        typeIntoPanel(store.dom.sourceInput, 'emod = 1\n# a long note\nx = 22\n');

        expect(painted()).toContainEqual(['# a long note', 'comment']);
      });

      it('leave text typed at the edge of a token plain, and keep the token', () => {
        const store = setupSourcePanel(source);
        answered(store);

        typeIntoPanel(store.dom.sourceInput, 'emod = 1\n# a note\nx = 222\n');

        expect(painted().slice(-2)).toEqual([['22', 'number'], ['2\n', null]]);
      });

      it('drop a token that the edit changes part of, so no colour lands on a character no answer read', () => {
        const store = setupSourcePanel(source);
        answered(store);

        typeIntoPanel(store.dom.sourceInput, 'emod = 1\n# a note\nx = 2y\n');

        expect(painted().slice(-2)).toEqual([['x', 'variable.member'], [' = 2y\n', null]]);
      });

      it('drop a token whose first character the edit replaces', () => {
        const store = setupSourcePanel(source);
        answered(store);

        typeIntoPanel(store.dom.sourceInput, 'emod = 1\n# a note\nx = 32\n');

        expect(painted().slice(-2)).toEqual([['x', 'variable.member'], [' = 32\n', null]]);
      });

      it("give a character typed inside a token that token's colour when it repeats the one before it", () => {
        const store = setupSourcePanel('x = 1b\n');
        SourcePanel.showTokens(store, 'x = 1b\n', [
          { start: 0, end: 1, category: 'variable.member' },
          { start: 4, end: 6, category: 'string' },
        ]);

        typeIntoPanel(store.dom.sourceInput, 'x = 11b\n');

        expect(painted()).toEqual([['x', 'variable.member'], [' = ', null], ['11b', 'string'], ['\n', null]]);
      });
    });

    it('draw the text a script writes into the panel, which fires no input event', () => {
      const store = setupSourcePanel('emod = 1\n');
      expect(coloursText()).toBe('emod = 1\n');

      store.dom.sourceInput.value = 'emod = 1\n\nactor "Clerk" {\n}\n';

      expect(coloursText()).toBe('emod = 1\n\nactor "Clerk" {\n}\n');
      expect(store.dom.sourceInput.value).toBe('emod = 1\n\nactor "Clerk" {\n}\n');
    });
  });

  describe('the text', () => {
    it('is announced on each change, typed or written by a script, and not when only its colours change', () => {
      const store = setupSourcePanel('emod = 1\n');
      const announced = [];
      const listen = ({ store: changed }) => announced.push(changed === store ? changed.dom.sourceInput.value : 'another panel');
      bus.on('source:changed', listen);

      try {
        typeIntoPanel(store.dom.sourceInput, 'emod = 2\n');
        store.dom.sourceInput.value = 'emod = 3\n';
        SourcePanel.showTokens(store, 'emod = 3\n', []);
      } finally {
        bus.off('source:changed', listen);
      }

      expect(announced).toEqual(['emod = 2\n', 'emod = 3\n']);
    });
  });

  describe('gutter', () => {
    it('marks each line that carries a diagnostic, and no other, naming its severity', () => {
      const store = setupSourcePanel();

      report(store, [
        { file: 'orders.emod', line: 7, column: 3, message: 'command "Ship" is never used', severity: 'warning' },
        { file: 'orders.emod', line: 3, column: 1, message: 'unexpected "}"', severity: 'error' },
      ]);

      expect(gutterEntries()).toEqual([
        { title: 'Line 3: 1 error', severity: 'error' },
        { title: 'Line 7: 1 warning', severity: 'warning' },
      ]);
    });

    it('shows the most severe on a line carrying more than one, counting a severity it does not know as an error', () => {
      const store = setupSourcePanel();

      report(store, [
        { file: 'orders.emod', line: 4, message: 'a', severity: 'info' },
        { file: 'orders.emod', line: 4, message: 'b', severity: 'warning' },
        { file: 'orders.emod', line: 9, message: 'c', severity: 'info' },
        { file: 'orders.emod', line: 9, message: 'd', severity: 'fatal' },
      ]);

      expect(gutterEntries()).toEqual([
        { title: 'Line 4: 1 warning, 1 info', severity: 'warning' },
        { title: 'Line 9: 1 error, 1 info', severity: 'error' },
      ]);
    });

    it('marks no line for a diagnostic that applies to the whole file', () => {
      const store = setupSourcePanel();

      report(store, [
        { file: 'orders.emod', line: 0, column: 0, message: 'the model declares no context', severity: 'error' },
        { file: 'orders.emod', line: 2, column: 1, message: 'unexpected "}"', severity: 'info' },
      ]);

      expect(gutterEntries()).toEqual([{ title: 'Line 2: 1 info', severity: 'info' }]);
    });

    it('replaces the lines of the previous report, and marks none for a report that is empty', () => {
      const store = setupSourcePanel();
      report(store, [{ file: 'orders.emod', line: 3, message: 'a', severity: 'error' }]);

      report(store, [{ file: 'orders.emod', line: 5, message: 'b', severity: 'warning' }]);
      expect(gutterEntries()).toEqual([{ title: 'Line 5: 1 warning', severity: 'warning' }]);

      report(store, []);
      expect(gutterEntries()).toEqual([]);
    });
  });

  describe('caret', () => {
    const ordersSource = 'emod = 1\n\nmodel "Orders" {\n  oops\n}\n';

    function caretIn(store) {
      const panel = store.dom.sourceInput;
      return { start: panel.selectionStart, end: panel.selectionEnd, focused: document.activeElement === panel };
    }

    it('puts the caret before the character a location names, with nothing selected, and gives the source panel focus', () => {
      const store = setupSourcePanel(ordersSource);

      SourcePanel.moveCaretTo(store, { line: 4, column: 3 });

      const oops = ordersSource.indexOf('oops');
      expect(caretIn(store)).toEqual({ start: oops, end: oops, focused: true });
    });

    it.each([0, undefined])('puts the caret at the start of the line when the column is %s, which is not known', (column) => {
      const store = setupSourcePanel(ordersSource);

      SourcePanel.moveCaretTo(store, { line: 4, column });

      expect(caretIn(store).start).toBe(ordersSource.indexOf('  oops'));
    });

    it('counts a column in the characters a reader sees, as HCL does, not in code units', () => {
      const source = 'emod = 1\nactor "Cáfe\u{1F469}‍\u{1F4BB}" { 7 }\n';
      const store = setupSourcePanel(source);

      SourcePanel.moveCaretTo(store, { line: 2, column: 17 });

      expect(caretIn(store).start).toBe(source.indexOf('7'));
    });

    it('stops at the end of the line when the column is past it', () => {
      const store = setupSourcePanel(ordersSource);

      SourcePanel.moveCaretTo(store, { line: 4, column: 40 });

      expect(caretIn(store).start).toBe(ordersSource.indexOf('oops') + 'oops'.length);
    });

    it('puts the caret at the end of the text when the line is past its end', () => {
      const store = setupSourcePanel(ordersSource);

      SourcePanel.moveCaretTo(store, { line: 40, column: 2 });

      expect(caretIn(store).start).toBe(ordersSource.length);
    });
  });

  describe('caret line', () => {
    const ordersSource = 'emod = 1\n\nmodel "Orders" {\n  oops\n}\n';

    function putCaretOnLine(store, line) {
      const offset = ordersSource.split('\n').slice(0, line - 1).join('\n').length + (line > 1 ? 1 : 0);
      store.dom.sourceInput.setSelectionRange(offset, offset);
      SourcePanel.showDiagnostics(store);
    }

    const shown = () => document.getElementById('caret-line-diagnostics').textContent;

    it("shows each diagnostic on the caret's line with its severity and rule, most severe first", () => {
      const store = setupSourcePanel(ordersSource);
      store.diagnostics = [
        { file: 'orders.emod', line: 4, column: 3, message: 'consider a longer name', severity: 'info', rule_name: 'short-name' },
        { file: 'orders.emod', line: 4, column: 3, message: 'unexpected "oops"', severity: 'error' },
        { file: 'orders.emod', line: 2, column: 1, message: 'a finding on another line', severity: 'warning' },
      ];

      putCaretOnLine(store, 4);

      expect(shown()).toContain('error');
      expect(shown()).toContain('[short-name] consider a longer name');
      expect(shown().indexOf('unexpected "oops"')).toBeLessThan(shown().indexOf('consider a longer name'));
      expect(shown()).not.toContain('a finding on another line');
    });

    it('gives each row a tooltip that reads as the row does, its severity apart from its message', () => {
      const store = setupSourcePanel(ordersSource);
      store.diagnostics = [
        { file: 'orders.emod', line: 4, column: 3, message: 'consider a longer name', severity: 'info', rule_name: 'short-name' },
        { file: 'orders.emod', line: 4, column: 3, message: 'unexpected "oops"', severity: 'error' },
      ];

      putCaretOnLine(store, 4);

      expect(Array.from(document.querySelectorAll('#caret-line-diagnostics [title]')).map((row) => row.title))
        .toEqual(['error unexpected "oops"', 'info [short-name] consider a longer name']);
    });

    it('clears when the caret moves to a line that carries no diagnostic', () => {
      const store = setupSourcePanel(ordersSource);
      store.diagnostics = [{ file: 'orders.emod', line: 4, column: 3, message: 'unexpected "oops"', severity: 'error' }];
      putCaretOnLine(store, 4);
      expect(shown()).toContain('unexpected "oops"');

      putCaretOnLine(store, 3);

      expect(shown()).toBe('');
    });

    it('leaves the message untouched when the caret moves within the same line, so it is not announced again', async () => {
      const store = setupSourcePanel(ordersSource);
      store.diagnostics = [{ file: 'orders.emod', line: 4, column: 3, message: 'unexpected "oops"', severity: 'error' }];
      putCaretOnLine(store, 4);
      const changes = [];
      new MutationObserver((records) => changes.push(...records))
        .observe(document.getElementById('caret-line-diagnostics'), { childList: true, subtree: true, characterData: true });

      const oops = ordersSource.indexOf('oops');
      store.dom.sourceInput.setSelectionRange(oops + 2, oops + 2);
      SourcePanel.showDiagnostics(store);
      await Promise.resolve();

      expect(changes).toHaveLength(0);
      expect(shown()).toContain('unexpected "oops"');
    });

    it.each([
      ['forward', 'unexpected "oops"', 'a finding on line 2'],
      ['backward', 'a finding on line 2', 'unexpected "oops"'],
    ])('takes the caret of a %s selection from the end it was extended to', (direction, onCaretLine, elsewhere) => {
      const store = setupSourcePanel(ordersSource);
      store.diagnostics = [
        { file: 'orders.emod', line: 2, column: 1, message: 'a finding on line 2', severity: 'warning' },
        { file: 'orders.emod', line: 4, column: 3, message: 'unexpected "oops"', severity: 'error' },
      ];

      store.dom.sourceInput.setSelectionRange(ordersSource.indexOf('\n') + 1, ordersSource.indexOf('oops'), direction);
      SourcePanel.showDiagnostics(store);

      expect(shown()).toContain(onCaretLine);
      expect(shown()).not.toContain(elsewhere);
    });

    it('never shows a diagnostic that applies to the whole file as the message of a line', () => {
      const store = setupSourcePanel(ordersSource);
      store.diagnostics = [
        { file: 'orders.emod', line: 0, column: 0, message: 'the model declares no context', severity: 'error' },
        { file: 'orders.emod', line: 1, column: 1, message: 'version 1 is old', severity: 'info' },
      ];

      putCaretOnLine(store, 1);

      expect(shown()).toContain('version 1 is old');
      expect(shown()).not.toContain('the model declares no context');
    });
  });
});
