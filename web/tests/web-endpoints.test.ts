import { expect, it } from 'vitest';
import { webEndpoints } from '../features/play/src/web-endpoints';

it('selects only runtime-published loopback HTTP entry points for problem frames', () => {
  const addresses = ['http://127.0.0.1:43123', 'http://127.0.0.1:65535/',
    'tcp://127.0.0.1:1234', 'https://other.example:1234', 'http://localhost:1234',
    'http://127.0.0.1:65536', 'http://127.0.0.1:0', 'http://127.0.0.1:1234/path',
    'http://user@127.0.0.1:1234', 'http://127.0.0.1:1234#secret', 'http://127.0.0.1:1234?redirect=other'];
  const endpoints = addresses.map((url, index) => ({ name: String(index), url }));
  expect(webEndpoints(endpoints)).toEqual(endpoints.slice(0, 2));
});
