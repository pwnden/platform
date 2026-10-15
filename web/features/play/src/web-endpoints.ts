import type { Endpoint } from '@pwnden/play';

// The platform supplies declared HTTP services on owned per-problem loopback origins (<slug>.localhost).
export function webEndpoints(endpoints: readonly Endpoint[]): readonly Endpoint[] {
  return endpoints.filter(endpoint => {
    const match = /^http:\/\/[a-z0-9]+(?:-[a-z0-9]+)*\.localhost:([1-9][0-9]{0,4})\/?$/.exec(endpoint.url);
    return match !== null && Number(match[1]) <= 65535;
  });
}
