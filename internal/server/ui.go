package server

var dashboardHTML = []byte(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Stockyard Billfold</title>
<style>
  :root {
    --bg: #1a1410;
    --surface: #241c15;
    --border: #3d2e1e;
    --rust: #c4622d;
    --leather: #8b5e3c;
    --cream: #f5e6c8;
    --muted: #7a6550;
    --text: #e8d5b0;
  }
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body { background: var(--bg); color: var(--text); font-family: 'JetBrains Mono', monospace, sans-serif; min-height: 100vh; }
  header { background: var(--surface); border-bottom: 1px solid var(--border); padding: 1rem 2rem; display: flex; align-items: center; gap: 1rem; }
  .logo { color: var(--rust); font-size: 1.25rem; font-weight: 700; letter-spacing: 0.05em; }
  .badge { background: var(--rust); color: var(--cream); font-size: 0.65rem; padding: 0.2rem 0.5rem; border-radius: 3px; font-weight: 600; text-transform: uppercase; }
  main { max-width: 960px; margin: 2rem auto; padding: 0 2rem; }
  .hero { text-align: center; padding: 3rem 0 2rem; }
  .hero h1 { font-size: 2rem; color: var(--cream); margin-bottom: 0.5rem; }
  .hero p { color: var(--muted); font-size: 0.95rem; max-width: 480px; margin: 0 auto; }
  .stats { display: grid; grid-template-columns: repeat(3, 1fr); gap: 1rem; margin: 2rem 0; }
  .stat { background: var(--surface); border: 1px solid var(--border); border-radius: 6px; padding: 1.25rem; text-align: center; }
  .stat-value { font-size: 1.75rem; font-weight: 700; color: var(--rust); }
  .stat-label { font-size: 0.75rem; color: var(--muted); margin-top: 0.25rem; text-transform: uppercase; letter-spacing: 0.05em; }
  .card { background: var(--surface); border: 1px solid var(--border); border-radius: 6px; padding: 1.5rem; margin-bottom: 1rem; }
  .card h2 { font-size: 1rem; color: var(--cream); margin-bottom: 1rem; }
  .tier-box { display: grid; grid-template-columns: 1fr 1fr; gap: 1rem; }
  .tier { background: var(--bg); border: 1px solid var(--border); border-radius: 4px; padding: 1rem; }
  .tier.pro { border-color: var(--rust); }
  .tier-name { font-size: 0.7rem; text-transform: uppercase; letter-spacing: 0.1em; color: var(--muted); margin-bottom: 0.5rem; }
  .tier.pro .tier-name { color: var(--rust); }
  .tier-desc { font-size: 0.85rem; color: var(--text); }
  .tier-price { font-size: 0.8rem; color: var(--leather); margin-top: 0.5rem; }
  footer { text-align: center; padding: 2rem; color: var(--muted); font-size: 0.75rem; }
  footer a { color: var(--leather); text-decoration: none; }
  .endpoint-table { width: 100%; border-collapse: collapse; font-size: 0.8rem; }
  .endpoint-table th { text-align: left; color: var(--muted); padding: 0.5rem; border-bottom: 1px solid var(--border); }
  .endpoint-table td { padding: 0.5rem; border-bottom: 1px solid var(--border); color: var(--text); }
  .method { color: var(--rust); font-weight: 600; }
</style>
</head>
<body>
<header>
  <span class="logo">⬡ Stockyard</span>
  <span style="color:var(--muted);">/</span>
  <span style="color:var(--cream);font-weight:600;">Billfold</span>
  <span class="badge">v0.1.0</span>
</header>
<main>
  <div class="hero">
    <h1>Billfold</h1>
    <p>Invoicing for freelancers — clients, PDF invoices, payment status, simple reports</p>
  </div>
  <div class="stats">
    <div class="stat">
      <div class="stat-value" id="stat-items">—</div>
      <div class="stat-label">Total Items</div>
    </div>
    <div class="stat">
      <div class="stat-value">9270</div>
      <div class="stat-label">Port</div>
    </div>
    <div class="stat">
      <div class="stat-value" id="stat-tier">—</div>
      <div class="stat-label">Tier</div>
    </div>
  </div>
  <div class="card">
    <h2>Tier &amp; Limits</h2>
    <div class="tier-box">
      <div class="tier">
        <div class="tier-name">Free</div>
        <div class="tier-desc">3 clients, 10 invoices</div>
        <div class="tier-price">$0/mo</div>
      </div>
      <div class="tier pro">
        <div class="tier-name">Pro</div>
        <div class="tier-desc">Unlimited clients and invoices</div>
        <div class="tier-price">$2.99/mo</div>
      </div>
    </div>
  </div>
  <div class="card">
    <h2>API Endpoints</h2>
    <table class="endpoint-table">
      <thead><tr><th>Method</th><th>Path</th><th>Description</th></tr></thead>
      <tbody>
        <tr><td class="method">GET</td><td>/health</td><td>Health check</td></tr>
        <tr><td class="method">GET</td><td>/api/version</td><td>Version info</td></tr>
        <tr><td class="method">GET</td><td>/api/limits</td><td>Current tier limits</td></tr>
        <tr><td class="method">GET</td><td>/api/items</td><td>List items</td></tr>
        <tr><td class="method">POST</td><td>/api/items</td><td>Create item</td></tr>
        <tr><td class="method">GET</td><td>/api/items/{id}</td><td>Get item</td></tr>
        <tr><td class="method">PUT</td><td>/api/items/{id}</td><td>Update item</td></tr>
        <tr><td class="method">DELETE</td><td>/api/items/{id}</td><td>Delete item</td></tr>
      </tbody>
    </table>
  </div>
</main>
<footer>
  <a href="https://stockyard.dev">stockyard.dev</a> &mdash; Creator & Small Business &mdash; Apache 2.0
</footer>
<script>
fetch('/api/limits').then(r=>r.json()).then(d=>{
  document.getElementById('stat-tier').textContent = d.tier.toUpperCase();
});
fetch('/api/items').then(r=>r.json()).then(d=>{
  document.getElementById('stat-items').textContent = Array.isArray(d) ? d.length : '0';
});
</script>
</body>
</html>`)
