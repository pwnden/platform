export type TerminalPlatform = 'windows' | 'macos' | 'linux';

export function terminalPlatform(browser: { platform: string; userAgentData?: { platform: string } }): TerminalPlatform {
  const platform = browser.userAgentData?.platform || browser.platform;
  if (/mac|iphone|ipad|ipod/i.test(platform)) return 'macos';
  return /win/i.test(platform) ? 'windows' : 'linux';
}

export function terminalClipboardKeys(
  platform: TerminalPlatform,
  selection: { hasSelection(): boolean },
  copy: () => void,
): (event: KeyboardEvent) => boolean {
  return event => {
    if (event.isComposing || event.altKey) return true;
    const key = event.key.toLowerCase();
    if (platform === 'macos') {
      // Native clipboard events preserve browser context-menu and paste behavior.
      return !(event.metaKey && !event.ctrlKey && !event.shiftKey && (key === 'c' || key === 'v'));
    }
    if (event.metaKey) return true;
    if (event.ctrlKey && event.shiftKey && key === 'c') {
      // Browsers otherwise reserve this chord for developer tools.
      event.preventDefault();
      if (event.type === 'keydown' && selection.hasSelection()) copy();
      return false;
    }
    if (event.ctrlKey && !event.shiftKey && key === 'c' && platform === 'windows' && selection.hasSelection()) return false;
    if (event.ctrlKey && key === 'v' && (event.shiftKey || platform === 'windows')) return false;
    if (key === 'insert' && ((event.ctrlKey && !event.shiftKey) || (event.shiftKey && !event.ctrlKey))) return false;
    return true;
  };
}

export async function copyTerminalSelection(document: Document, clipboard: Pick<Clipboard, 'writeText'> | undefined, text: string): Promise<boolean> {
  if (!text) return true;
  let copied = false;
  const copy = (event: ClipboardEvent) => {
    if (!event.clipboardData) return;
    event.clipboardData.setData('text/plain', text);
    event.preventDefault();
    copied = true;
  };
  // Use the synchronous user gesture first; this does not require clipboard read permission.
  const options = { capture: true };
  document.addEventListener('copy', copy, options);
  try {
    if (document.execCommand('copy') && copied) return true;
  } catch { /* Use the Clipboard API when the browser has no copy command. */ }
  finally { document.removeEventListener('copy', copy, options); }
  try {
    if (!clipboard) return false;
    await clipboard.writeText(text);
    return true;
  } catch { return false; }
}
