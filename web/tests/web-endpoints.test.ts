import { expect, it } from 'vitest';
import { webEndpoints } from '../features/play/src/web-endpoints';

it('selects only per-problem loopback HTTP entry points for problem frames', () => {
  const addresses = ['http://note-vault.localhost:43123', 'http://note-vault.localhost:65535/',
    'http://127.0.0.1:43123', 'tcp://note-vault.localhost:1234', 'https://other.example:1234', 'http://localhost:1234',
    'http://note-vault.localhost:65536', 'http://note-vault.localhost:0', 'http://note-vault.localhost:1234/path',
    'http://user@note-vault.localhost:1234', 'http://note-vault.localhost:1234#secret', 'http://note-vault.localhost:1234?redirect=other',
    'http://a.b.localhost:1234', 'http://-bad.localhost:1234', 'http://note_vault.localhost:1234'];
  const endpoints = addresses.map((url, index) => ({ name: String(index), url }));
  expect(webEndpoints(endpoints)).toEqual(endpoints.slice(0, 2));
});
