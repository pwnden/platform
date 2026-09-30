// Scope nonce assignment to xterm's documented documentOverride. Native DOM
// methods retain their real receiver; the global document is never modified.
export function terminalDocument(document: Document): Document {
  const nonce = document.querySelector<HTMLMetaElement>('meta[name="pwnden-style-nonce"]')?.content;
  if (!nonce || !/^[a-f0-9]{64}$/.test(nonce)) throw new Error('Missing terminal style nonce.');
  return new Proxy(document, {
    get(target, property) {
      if (property === 'createElement') {
        return (name: string, options?: ElementCreationOptions) => {
          const element = target.createElement(name, options);
          if (name.toLowerCase() === 'style') element.setAttribute('nonce', nonce);
          // xterm 6's viewport creates styles through the native document even
          // with documentOverride. Assign the nonce before insertion into its
          // own containers; leave global document and DOM prototypes untouched.
          if (name.toLowerCase() === 'div') {
            const append = element.appendChild;
            element.appendChild = function<T extends Node>(node: T): T {
              if (node.nodeName === 'STYLE') (node as unknown as HTMLStyleElement).setAttribute('nonce', nonce);
              return append.call(this, node) as T;
            };
          }
          return element;
        };
      }
      const value: unknown = Reflect.get(target, property, target);
      return typeof value === 'function' ? value.bind(target) : value;
    },
  });
}
