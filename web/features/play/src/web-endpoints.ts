import type { Endpoint } from '@pwnden/play';

// The runtime publishes challenge services on IPv4 loopback with assigned ports.
export function webEndpoints(endpoints: readonly Endpoint[]): readonly Endpoint[] {
  return endpoints.filter(endpoint => {
    const match = /^http:\/\/127\.0\.0\.1:([1-9][0-9]{0,4})\/?$/.exec(endpoint.url);
    return match !== null && Number(match[1]) <= 65535;
  });
}
