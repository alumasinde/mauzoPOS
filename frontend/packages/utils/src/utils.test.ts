import { describe, expect, it, vi } from 'vitest';
import { formatMoney } from './money';
import { matches, bindShortcuts } from './keyboard';

describe('formatMoney', () => {
  it.each([
    ['750.34', 'KSh 750.34'],
    ['1234567.5', 'KSh 1,234,567.50'],
    ['0.125', 'KSh 0.13'],
    ['99.995', 'KSh 100.00'],
    ['-90', '-KSh 90.00'],
    ['-0.001', 'KSh 0.00'],
    ['0.1', 'KSh 0.10'],
  ])('%s → %s', (input, want) => {
    expect(formatMoney(input)).toBe(want);
  });

  it('supports no symbol and leaves junk untouched', () => {
    expect(formatMoney('660', '')).toBe('660.00');
    expect(formatMoney('abc')).toBe('abc');
  });
});

describe('keyboard', () => {
  it('matches modifiers exactly', () => {
    const e = new KeyboardEvent('keydown', { key: 'F5' });
    expect(matches(e, { key: 'f5', handler: () => {} })).toBe(true);
    expect(matches(e, { key: 'F5', ctrl: true, handler: () => {} })).toBe(false);
  });

  it('binds and unbinds', () => {
    const handler = vi.fn();
    const el = document.createElement('div');
    const off = bindShortcuts([{ key: 'Escape', handler }], el);
    el.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
    off();
    el.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
    expect(handler).toHaveBeenCalledTimes(1);
  });
});
