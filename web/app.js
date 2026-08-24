async function load() {
  try {
    const [health, status, system] = await Promise.all([
      fetch('/api/v1/health').then(r => r.json()),
      fetch('/api/v1/status').then(r => r.json()),
      fetch('/api/v1/system').then(r => r.json())
    ]);
    document.querySelector('#health').textContent = health.ok ? 'CORE ONLINE' : 'ERROR';
    document.querySelector('#core-state').textContent = status.state.toUpperCase();
    document.querySelector('#network-state').textContent = status.network.connected ? 'CONNECTED' : 'OFFLINE';
    document.querySelector('#vocoder-state').textContent = status.vocoder.available ? 'READY' : 'NONE';
    document.querySelector('#version').textContent = `${system.version} · ${system.goarch}/${system.goos}`;
  } catch (error) {
    document.querySelector('#health').textContent = 'CORE OFFLINE';
    console.error(error);
  }
}
load();
