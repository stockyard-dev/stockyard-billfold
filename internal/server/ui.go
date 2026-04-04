package server
import "net/http"
func(s *Server)dashboard(w http.ResponseWriter,r *http.Request){w.Header().Set("Content-Type","text/html");w.Write([]byte(dashHTML))}
const dashHTML=`<!DOCTYPE html><html><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1.0"><title>Billfold</title>
<style>:root{--bg:#1a1410;--bg2:#241e18;--bg3:#2e261e;--rust:#e8753a;--leather:#a0845c;--cream:#f0e6d3;--cd:#bfb5a3;--cm:#7a7060;--gold:#d4a843;--green:#4a9e5c;--red:#c94444;--orange:#d4843a;--mono:'JetBrains Mono',monospace}
*{margin:0;padding:0;box-sizing:border-box}body{background:var(--bg);color:var(--cream);font-family:var(--mono);line-height:1.5}
.hdr{padding:1rem 1.5rem;border-bottom:1px solid var(--bg3);display:flex;justify-content:space-between;align-items:center}.hdr h1{font-size:.9rem;letter-spacing:2px}
.main{padding:1.5rem;max-width:900px;margin:0 auto}
.stats{display:grid;grid-template-columns:repeat(3,1fr);gap:.5rem;margin-bottom:1.2rem}
.st{background:var(--bg2);border:1px solid var(--bg3);padding:.7rem;text-align:center}.st-v{font-size:1.2rem}.st-l{font-size:.5rem;color:var(--cm);text-transform:uppercase;letter-spacing:1px;margin-top:.1rem}
.tabs{display:flex;gap:.3rem;margin-bottom:1rem}
.tab{font-size:.6rem;padding:.25rem .6rem;border:1px solid var(--bg3);background:var(--bg);color:var(--cm);cursor:pointer}.tab:hover{border-color:var(--leather)}.tab.active{border-color:var(--rust);color:var(--rust)}
.inv{background:var(--bg2);border:1px solid var(--bg3);padding:.8rem 1rem;margin-bottom:.5rem}
.inv-top{display:flex;justify-content:space-between;align-items:center}
.inv-name{font-size:.82rem;color:var(--cream)}
.inv-amount{font-size:.9rem}
.inv-meta{font-size:.6rem;color:var(--cm);margin-top:.2rem;display:flex;gap:.8rem}
.badge-draft{background:#d4a84322;color:var(--gold);border:1px solid #d4a84344;font-size:.5rem;padding:.1rem .3rem;text-transform:uppercase;letter-spacing:1px}
.badge-sent{background:#4a7ec922;color:#4a7ec9;border:1px solid #4a7ec944;font-size:.5rem;padding:.1rem .3rem;text-transform:uppercase;letter-spacing:1px}
.badge-paid{background:#4a9e5c22;color:var(--green);border:1px solid #4a9e5c44;font-size:.5rem;padding:.1rem .3rem;text-transform:uppercase;letter-spacing:1px}
.badge-overdue{background:#c9444422;color:var(--red);border:1px solid #c9444444;font-size:.5rem;padding:.1rem .3rem;text-transform:uppercase;letter-spacing:1px}
.btn{font-size:.6rem;padding:.25rem .6rem;cursor:pointer;border:1px solid var(--bg3);background:var(--bg);color:var(--cd)}.btn:hover{border-color:var(--leather);color:var(--cream)}
.btn-p{background:var(--rust);border-color:var(--rust);color:var(--bg)}
.modal-bg{display:none;position:fixed;inset:0;background:rgba(0,0,0,.6);z-index:100;align-items:center;justify-content:center}.modal-bg.open{display:flex}
.modal{background:var(--bg2);border:1px solid var(--bg3);padding:1.5rem;width:420px;max-width:90vw}
.modal h2{font-size:.8rem;margin-bottom:1rem;color:var(--rust)}
.fr{margin-bottom:.5rem}.fr label{display:block;font-size:.55rem;color:var(--cm);text-transform:uppercase;letter-spacing:1px;margin-bottom:.15rem}
.fr input,.fr select,.fr textarea{width:100%;padding:.35rem .5rem;background:var(--bg);border:1px solid var(--bg3);color:var(--cream);font-family:var(--mono);font-size:.7rem}
.acts{display:flex;gap:.4rem;justify-content:flex-end;margin-top:.8rem}
.empty{text-align:center;padding:3rem;color:var(--cm);font-style:italic;font-size:.75rem}
</style></head><body>
<div class="hdr"><h1>BILLFOLD</h1><button class="btn btn-p" onclick="openForm()">+ New Invoice</button></div>
<div class="main">
<div class="stats" id="stats"></div>
<div class="tabs" id="tabs"></div>
<div id="invoices"></div>
</div>
<div class="modal-bg" id="mbg" onclick="if(event.target===this)cm()"><div class="modal" id="mdl"></div></div>
<script>
const A='/api';let invoices=[],filter='all';
async function load(){const r=await fetch(A+'/invoices').then(r=>r.json());invoices=r.invoices||[];
const total=invoices.reduce((s,i)=>s+i.amount,0);const paid=invoices.filter(i=>i.status==='paid').reduce((s,i)=>s+i.amount,0);const outstanding=total-paid;
document.getElementById('stats').innerHTML='<div class="st"><div class="st-v">$'+fmt(total/100)+'</div><div class="st-l">Total</div></div><div class="st"><div class="st-v" style="color:var(--green)">$'+fmt(paid/100)+'</div><div class="st-l">Paid</div></div><div class="st"><div class="st-v" style="color:var(--orange)">$'+fmt(outstanding/100)+'</div><div class="st-l">Outstanding</div></div>';
document.getElementById('tabs').innerHTML=['all','draft','sent','paid','overdue'].map(t=>'<button class="tab'+(filter===t?' active':'')+'" onclick="setFilter(\''+t+'\')">'+t+'</button>').join('');
render();}
function setFilter(f){filter=f;render();}
function render(){let filtered=filter==='all'?invoices:invoices.filter(i=>i.status===filter);
if(!filtered.length){document.getElementById('invoices').innerHTML='<div class="empty">No invoices.</div>';return;}
let h='';filtered.forEach(i=>{
h+='<div class="inv"><div class="inv-top"><div class="inv-name">'+esc(i.name)+'</div><div style="display:flex;gap:.5rem;align-items:center"><span class="inv-amount">$'+(i.amount/100).toFixed(2)+'</span><span class="badge-'+i.status+'">'+i.status+'</span></div></div>';
h+='<div class="inv-meta">';if(i.due_date)h+='<span>Due: '+i.due_date+'</span>';h+='<span>'+ft(i.created_at)+'</span>';
if(i.paid_at)h+='<span>Paid: '+ft(i.paid_at)+'</span>';
h+='</div><div style="display:flex;gap:.3rem;margin-top:.4rem">';
if(i.status==='draft')h+='<button class="btn" onclick="setStatus(\''+i.id+'\',\'sent\')">Mark Sent</button>';
if(i.status==='sent'||i.status==='overdue')h+='<button class="btn" onclick="setStatus(\''+i.id+'\',\'paid\')">Mark Paid</button>';
h+='<button class="btn" onclick="del(\''+i.id+'\')" style="font-size:.5rem;color:var(--cm)">✕</button></div></div>';});
document.getElementById('invoices').innerHTML=h;}
async function setStatus(id,status){await fetch(A+'/invoices/'+id,{method:'PUT',headers:{'Content-Type':'application/json'},body:JSON.stringify({status})});load();}
async function del(id){if(confirm('Delete?')){await fetch(A+'/invoices/'+id,{method:'DELETE'});load();}}
function openForm(){document.getElementById('mdl').innerHTML='<h2>New Invoice</h2><div class="fr"><label>Client / Description</label><input id="f-n" placeholder="e.g. Acme Corp — March retainer"></div><div class="fr"><label>Amount ($)</label><input id="f-a" type="number" step="0.01" placeholder="500.00"></div><div class="fr"><label>Due Date</label><input id="f-d" type="date"></div><div class="fr"><label>Notes</label><textarea id="f-nt" rows="2"></textarea></div><div class="acts"><button class="btn" onclick="cm()">Cancel</button><button class="btn btn-p" onclick="sub()">Create</button></div>';document.getElementById('mbg').classList.add('open');}
async function sub(){const amt=Math.round(parseFloat(document.getElementById('f-a').value)*100);await fetch(A+'/invoices',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({name:document.getElementById('f-n').value,amount:amt,due_date:document.getElementById('f-d').value,notes:document.getElementById('f-nt').value})});cm();load();}
function cm(){document.getElementById('mbg').classList.remove('open');}
function fmt(n){return n.toFixed(2);}
function ft(t){if(!t)return'';return new Date(t).toLocaleDateString();}
function esc(s){if(!s)return'';const d=document.createElement('div');d.textContent=s;return d.innerHTML;}
load();
</script></body></html>`
