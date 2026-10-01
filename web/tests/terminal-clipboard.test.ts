import { expect, it, vi } from 'vitest';
import { copyTerminalSelection, terminalClipboardKeys, terminalPlatform } from '../packages/ui/src/terminal-clipboard';

function key(key: string, modifiers: Partial<KeyboardEvent> = {}): KeyboardEvent {
  return { type: 'keydown', key, ctrlKey: false, metaKey: false, altKey: false, shiftKey: false, isComposing: false, preventDefault: vi.fn(), ...modifiers } as KeyboardEvent;
}

it('detects the browser platform, including a Windows browser used with a Linux server', () => {
  expect(terminalPlatform({ platform: 'Linux x86_64', userAgentData: { platform: 'Windows' } })).toBe('windows');
  expect(terminalPlatform({ platform: 'Win32' })).toBe('windows');
  expect(terminalPlatform({ platform: 'MacIntel' })).toBe('macos');
  expect(terminalPlatform({ platform: 'iPad' })).toBe('macos');
  expect(terminalPlatform({ platform: 'Linux x86_64' })).toBe('linux');
});

it('copies a Windows selection with Ctrl+C and sends Ctrl+C to the shell when nothing is selected', () => {
  let selected = true;
  const copy = vi.fn();
  const handle = terminalClipboardKeys('windows', { hasSelection: () => selected }, copy);
  const event = key('c', { ctrlKey: true });
  expect(handle(event)).toBe(false);
  expect(event.preventDefault).not.toHaveBeenCalled(); // Browser copy event reaches xterm.
  expect(copy).not.toHaveBeenCalled();
  selected = false;
  expect(handle(event)).toBe(true); // xterm sends ETX to the shell.
});

it.each(['windows', 'linux'] as const)('copies only once with Ctrl+Shift+C on %s and prevents browser developer tools', platform => {
  const copy = vi.fn();
  const handle = terminalClipboardKeys(platform, { hasSelection: () => true }, copy);
  const event = key('C', { ctrlKey: true, shiftKey: true });
  expect(handle(event)).toBe(false);
  expect(event.preventDefault).toHaveBeenCalledOnce();
  expect(handle(key('C', { ...event, type: 'keypress' }))).toBe(false);
  expect(handle(key('C', { ...event, type: 'keyup' }))).toBe(false);
  expect(copy).toHaveBeenCalledOnce();
});

it('consumes the explicit copy shortcut without sending a shell interrupt when no text is selected', () => {
  const copy = vi.fn();
  const handle = terminalClipboardKeys('windows', { hasSelection: () => false }, copy);
  const event = key('C', { ctrlKey: true, shiftKey: true });
  expect(handle(event)).toBe(false);
  expect(event.preventDefault).toHaveBeenCalledOnce();
  expect(copy).not.toHaveBeenCalled();
});

it.each(['macos', 'linux'] as const)('preserves Ctrl+C interruption even with a selection on %s', platform => {
  const copy = vi.fn();
  expect(terminalClipboardKeys(platform, { hasSelection: () => true }, copy)(key('c', { ctrlKey: true }))).toBe(true);
  expect(copy).not.toHaveBeenCalled();
});

it('uses native Command+C/V on macOS and preserves Control+V for the shell', () => {
  const copy = vi.fn();
  const handle = terminalClipboardKeys('macos', { hasSelection: () => true }, copy);
  for (const value of ['c', 'v']) {
    const event = key(value, { metaKey: true });
    expect(handle(event)).toBe(false);
    expect(event.preventDefault).not.toHaveBeenCalled();
  }
  expect(handle(key('v', { ctrlKey: true }))).toBe(true);
  expect(copy).not.toHaveBeenCalled();
});

it('delegates Windows and Linux paste chords to native paste events so xterm owns bracketed paste', () => {
  const selection = { hasSelection: () => true };
  const windows = terminalClipboardKeys('windows', selection, vi.fn());
  const linux = terminalClipboardKeys('linux', selection, vi.fn());
  for (const handle of [windows, linux]) {
    for (const event of [key('V', { ctrlKey: true, shiftKey: true }), key('Insert', { shiftKey: true })]) {
      expect(handle(event)).toBe(false);
      expect(event.preventDefault).not.toHaveBeenCalled();
    }
    expect(handle(key('Insert', { ctrlKey: true }))).toBe(false);
  }
  expect(windows(key('v', { ctrlKey: true }))).toBe(false);
  expect(linux(key('v', { ctrlKey: true }))).toBe(true);
});

it('leaves IME composition, Alt/AltGr chords and other shell editing keys to xterm', () => {
  const copy = vi.fn();
  const handle = terminalClipboardKeys('windows', { hasSelection: () => true }, copy);
  for (const event of [
    key('C', { ctrlKey: true, shiftKey: true, isComposing: true }),
    key('c', { ctrlKey: true, altKey: true }), key('c', { ctrlKey: true, metaKey: true }),
    key('Tab'), key('ArrowUp'), key('a', { ctrlKey: true }), key('r', { ctrlKey: true }), key('z', { ctrlKey: true }),
  ]) {
    expect(handle(event)).toBe(true);
    expect(event.preventDefault).not.toHaveBeenCalled();
  }
  expect(copy).not.toHaveBeenCalled();
});

function copyDocument(command: (target: EventTarget) => boolean) {
  const target = new EventTarget();
  const document = {
    addEventListener: target.addEventListener.bind(target), removeEventListener: target.removeEventListener.bind(target),
    execCommand: vi.fn((action: string) => { expect(action).toBe('copy'); return command(target); }),
  } as unknown as Document;
  return { document, target };
}

it('copies selected Unicode text as plain text through the native copy event and removes its listener', async () => {
  const setData = vi.fn();
  const event = Object.assign(new Event('copy', { cancelable: true }), { clipboardData: { setData } });
  const { document, target } = copyDocument(target => { target.dispatchEvent(event); return true; });
  const clipboard = { writeText: vi.fn() };
  expect(await copyTerminalSelection(document, clipboard, '한글\n$ pwd')).toBe(true);
  expect(setData).toHaveBeenCalledWith('text/plain', '한글\n$ pwd');
  expect(event.defaultPrevented).toBe(true);
  expect(clipboard.writeText).not.toHaveBeenCalled();
  setData.mockClear();
  target.dispatchEvent(event);
  expect(setData).not.toHaveBeenCalled();
});

it('falls back to clipboard write without requesting read permission when native copy is unavailable', async () => {
  const { document } = copyDocument(() => { throw new Error('unsupported'); });
  const clipboard = { writeText: vi.fn(async () => {}) };
  expect(await copyTerminalSelection(document, clipboard, 'selected')).toBe(true);
  expect(clipboard.writeText).toHaveBeenCalledWith('selected');
});

it('reports denied or unavailable clipboard access and removes failed copy listeners', async () => {
  const { document, target } = copyDocument(target => { target.dispatchEvent(new Event('copy')); return true; });
  const clipboard = { writeText: vi.fn(async () => { throw new Error('denied'); }) };
  expect(await copyTerminalSelection(document, clipboard, 'selected')).toBe(false);
  expect(await copyTerminalSelection(document, undefined, 'selected')).toBe(false);
  const setData = vi.fn();
  target.dispatchEvent(Object.assign(new Event('copy'), { clipboardData: { setData } }));
  expect(setData).not.toHaveBeenCalled();
  expect(await copyTerminalSelection(document, clipboard, '')).toBe(true);
  expect(clipboard.writeText).toHaveBeenCalledOnce();
});
