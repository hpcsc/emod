import { describe, it, expect, beforeEach } from 'vitest';
import { SourcePanel } from '../static/source-panel.js';
import { createStore } from '../static/store.js';

function setupSourcePanel(text) {
  document.body.innerHTML = `
    <div id="source-gutter"></div>
    <textarea id="source-input"></textarea>
  `;
  const store = createStore();
  store.dom.sourceGutter = document.getElementById('source-gutter');
  store.dom.sourceInput = document.getElementById('source-input');
  store.dom.sourceInput.value = text || '';
  SourcePanel.initSourcePanel(store);
  return store;
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

describe('SourcePanel', () => {
  describe('gutter', () => {
    it('marks each line that carries a diagnostic, and no other, naming its severity', () => {
      const store = setupSourcePanel();

      SourcePanel.showGutter(store, [
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

      SourcePanel.showGutter(store, [
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

      SourcePanel.showGutter(store, [
        { file: 'orders.emod', line: 0, column: 0, message: 'the model declares no context', severity: 'error' },
        { file: 'orders.emod', line: 2, column: 1, message: 'unexpected "}"', severity: 'info' },
      ]);

      expect(gutterEntries()).toEqual([{ title: 'Line 2: 1 info', severity: 'info' }]);
    });

    it('replaces the lines of the previous report, and marks none for a report that is empty', () => {
      const store = setupSourcePanel();
      SourcePanel.showGutter(store, [{ file: 'orders.emod', line: 3, message: 'a', severity: 'error' }]);

      SourcePanel.showGutter(store, [{ file: 'orders.emod', line: 5, message: 'b', severity: 'warning' }]);
      expect(gutterEntries()).toEqual([{ title: 'Line 5: 1 warning', severity: 'warning' }]);

      SourcePanel.showGutter(store, []);
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
});
