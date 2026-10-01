import type { Endpoint } from '@pwnden/play';

// The platform supplies declared HTTP services on owned IPv4 loopback origins.
export function webEndpoints(endpoints: readonly Endpoint[]): readonly Endpoint[] {
  return endpoints.filter(endpoint => {
    const match = /^http:\/\/127\.0\.0\.1:([1-9][0-9]{0,4})\/?$/.exec(endpoint.url);
    return match !== null && Number(match[1]) <= 65535;
  });
}
