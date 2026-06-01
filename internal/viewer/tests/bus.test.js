import { describe, it, expect } from 'vitest';
import { bus } from '../static/bus.js';

describe('bus', () => {
  describe('on and emit', () => {
    it('calls subscriber with emitted data', () => {
      const calls = [];
      bus.on('test:event', (data) => calls.push(data));
      bus.emit('test:event', { value: 1 });
      expect(calls).toEqual([{ value: 1 }]);
    });

    it('calls multiple subscribers for the same event', () => {
      const a = [], b = [];
      bus.on('multi', (d) => a.push(d));
      bus.on('multi', (d) => b.push(d));
      bus.emit('multi', { n: 2 });
      expect(a).toEqual([{ n: 2 }]);
      expect(b).toEqual([{ n: 2 }]);
    });

    it('passes no data when emit has no payload', () => {
      const calls = [];
      bus.on('no-payload', (d) => calls.push(d));
      bus.emit('no-payload');
      expect(calls).toEqual([undefined]);
    });

    it('does not call subscriber for a different event', () => {
      const calls = [];
      bus.on('event:a', (d) => calls.push(d));
      bus.emit('event:b', { n: 1 });
      expect(calls).toEqual([]);
    });

    it('does nothing when emitting on unregistered event', () => {
      expect(() => bus.emit('ghost', {})).not.toThrow();
    });
  });

  describe('off', () => {
    it('removes a specific subscriber', () => {
      const calls = [];
      const fn = (d) => calls.push(d);
      bus.on('removable', fn);
      bus.emit('removable', { v: 1 });
      bus.off('removable', fn);
      bus.emit('removable', { v: 2 });
      expect(calls).toEqual([{ v: 1 }]);
    });

    it('does not affect other subscribers on the same event', () => {
      const a = [], b = [];
      const fnA = (d) => a.push(d);
      const fnB = (d) => b.push(d);
      bus.on('shared', fnA);
      bus.on('shared', fnB);
      bus.off('shared', fnA);
      bus.emit('shared', { n: 1 });
      expect(a).toEqual([]);
      expect(b).toEqual([{ n: 1 }]);
    });

    it('does nothing when unregistering a non-existent subscriber', () => {
      expect(() => bus.off('any', () => {})).not.toThrow();
    });

    it('does nothing when unregistering from a non-existent event', () => {
      expect(() => bus.off('never-registered', () => {})).not.toThrow();
    });
  });

  describe('isolation between events', () => {
    it('keeps subscriber lists independent', () => {
      const a = [], b = [];
      bus.on('evt:x', (d) => a.push(d));
      bus.on('evt:y', (d) => b.push(d));
      bus.emit('evt:x', { letter: 'x' });
      bus.emit('evt:y', { letter: 'y' });
      expect(a).toEqual([{ letter: 'x' }]);
      expect(b).toEqual([{ letter: 'y' }]);
    });
  });
});
