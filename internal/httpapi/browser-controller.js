(() => {
  const frame = document.getElementById('problem');
  const parentOrigin = document.body.dataset.parent;
  const targetOrigin = document.body.dataset.target;
  const channel = location.pathname;
  let loading = true;
  const observed = new WeakSet();
  let previous = '';
  function state() {
    let url = targetOrigin + '/';
    let canBack = false;
    let canForward = false;
    let error = '';
    try {
      const child = frame.contentWindow;
      const current = new URL(child.location.href);
      if (current.origin !== location.origin) throw new Error('outside target');
      url = targetOrigin + current.pathname + current.search + current.hash;
      const navigation = child.navigation;
      if (!navigation) {
        error = '이 브라우저는 내장 웹 탐색을 지원하지 않습니다. 새 탭에서 열어주세요.';
      } else {
        canBack = navigation.canGoBack;
        canForward = navigation.canGoForward;
        if (!observed.has(navigation)) {
          observed.add(navigation);
          navigation.addEventListener('navigate', event => { if (event.downloadRequest != null) return; loading = true; state(); });
          navigation.addEventListener('currententrychange', () => { loading = false; state(); });
          navigation.addEventListener('navigatesuccess', () => { loading = false; state(); });
          navigation.addEventListener('navigateerror', () => { loading = false; state(); });
        }
      }
    } catch {
      if (!loading) error = '문제 사이트의 보안 설정으로 내장 탐색을 사용할 수 없습니다. 새 탭에서 열어주세요.';
    }
    const payload = { type: 'pwnden.browser.state.v1', channel, url, canBack, canForward, busy: loading, error };
    const encoded = JSON.stringify(payload);
    if (encoded !== previous) { previous = encoded; parent.postMessage(payload, parentOrigin); }
  }
  window.addEventListener('message', event => {
    if (event.origin !== parentOrigin || event.source !== parent || event.data?.type !== 'pwnden.browser.command.v1' || event.data.channel !== channel) return;
    const action = event.data.action;
    try {
      const child = frame.contentWindow;
      const navigation = child.navigation;
      let result;
      if (action === 'back' && navigation?.canGoBack) result = navigation.back();
      else if (action === 'forward' && navigation?.canGoForward) result = navigation.forward();
      else if (action === 'reload') { loading = true; child.location.reload(); }
      else if (action === 'navigate' && typeof event.data.url === 'string') {
        const url = new URL(event.data.url, targetOrigin);
        if (url.origin !== targetOrigin || url.username || url.password) return;
        loading = true;
        child.location.assign(location.origin + url.pathname + url.search + url.hash);
      }
      if (result) result.finished.catch(() => { loading = false; state(); });
    } catch { loading = false; }
    state();
  });
  frame.addEventListener('load', () => { loading = false; state(); });
  frame.contentWindow.location.replace(location.origin + '/');
  const timer = setInterval(state, 250);
  window.addEventListener('pagehide', () => clearInterval(timer), { once: true });
  state();
})();
