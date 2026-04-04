package server

import "net/http"

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(dashHTML))
}

const dashHTML = `<!DOCTYPE html><html><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1.0"><title>Billfold</title>
<link href="https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500;700&display=swap" rel="stylesheet">
<style>
:root{--bg:#1a1410;--bg2:#241e18;--bg3:#2e261e;--rust:#e8753a;--leather:#a0845c;--cream:#f0e6d3;--cd:#bfb5a3;--cm:#7a7060;--gold:#d4a843;--green:#4a9e5c;--red:#c94444;--blue:#5b8dd9;--mono:'JetBrains Mono',monospace;--serif:'Libre Baskerville',serif}
*{margin:0;padding:0;box-sizing:border-box}body{background:var(--bg);color:var(--cream);font-family:var(--mono);line-height:1.5}
.hdr{padding:1rem 1.5rem;border-bottom:1px solid var(--bg3);display:flex;justify-content:space-between;align-items:center}.hdr h1{font-size:.9rem;letter-spacing:2px}.hdr h1 span{color:var(--rust)}
.main{padding:1.5rem;max-width:960px;margin:0 auto}
.stats{display:grid;grid-template-columns:repeat(4,1fr);gap:.5rem;margin-bottom:1rem}
.st{background:var(--bg2);border:1px solid var(--bg3);padding:.6rem;text-align:center;cursor:pointer;transition:border-color .2s}
.st:hover,.st.active{border-color:var(--rust)}.st.active .st-v{color:var(--rust)}
.st-v{font-size:1.2rem;font-weight:700}.st-l{font-size:.5rem;color:var(--cm);text-transform:uppercase;letter-spacing:1px;margin-top:.15rem}
.toolbar{display:flex;gap:.5rem;margin-bottom:1rem;align-items:center;flex-wrap:wrap}
.search{flex:1;min-width:180px;padding:.4rem .6rem;background:var(--bg2);border:1px solid var(--bg3);color:var(--cream);font-family:var(--mono);font-size:.7rem}
.search:focus{outline:none;border-color:var(--leather)}
.tabs{display:flex;gap:.3rem}
.tab{font-size:.55rem;padding:.25rem .5rem;border:1px solid var(--bg3);background:var(--bg);color:var(--cm);cursor:pointer}.tab:hover{border-color:var(--leather)}.tab.active{border-color:var(--rust);color:var(--rust)}
.inv{background:var(--bg2);border:1px solid var(--bg3);padding:.8rem 1rem;margin-bottom:.5rem;transition:border-color .2s}
.inv:hover{border-color:var(--leather)}
.inv-top{display:flex;justify-content:space-between;align-items:center;gap:.5rem}
.inv-name{font-size:.82rem;flex:1}
.inv-amount{font-size:.95rem;font-weight:700}
.inv-meta{font-size:.55rem;color:var(--cm);margin-top:.3rem;display:flex;gap:.7rem;align-items:center;flex-wrap:wrap}
.inv-notes{font-size:.65rem;color:var(--cm);margin-top:.3rem;font-style:italic;padding:.3rem .5rem;border-left:2px solid var(--bg3)}
.inv-actions{display:flex;gap:.3rem;margin-top:.4rem}
.badge{font-size:.5rem;padding:.15rem .4rem;text-transform:uppercase;letter-spacing:1px;border:1px solid;flex-shrink:0}
.badge.draft{border-color:var(--gold);color:var(--gold);background:#d4a84315}
.badge.sent{border-color:var(--blue);color:var(--blue);background:#5b8dd915}
.badge.paid{border-color:var(--green);color:var(--green);background:#4a9e5c15}
.badge.overdue{border-color:var(--red);color:var(--red);background:#c9444415}
.due-warn{color:var(--red)}
.btn{font-size:.6rem;padding:.25rem .5rem;cursor:pointer;border:1px solid var(--bg3);background:var(--bg);color:var(--cd);transition:all .2s}
.btn:hover{border-color:var(--leather);color:var(--cream)}.btn-p{background:var(--rust);border-color:var(--rust);color:#fff}.btn-p:hover{background:#d4682f}
.btn-green{background:var(--green);border-color:var(--green);color:#fff}
.btn-sm{font-size:.5rem;padding:.2rem .4rem}
.modal-bg{display:none;position:fixed;inset:0;background:rgba(0,0,0,.65);z-index:100;align-items:center;justify-content:center}.modal-bg.open{display:flex}
.modal{background:var(--bg2);border:1px solid var(--bg3);padding:1.5rem;width:460px;max-width:92vw;max-height:90vh;overflow-y:auto}
.modal h2{font-size:.8rem;margin-bottom:1rem;color:var(--rust);letter-spacing:1px}
.fr{margin-bottom:.6rem}.fr label{display:block;font-size:.55rem;color:var(--cm);text-transform:uppercase;letter-spacing:1px;margin-bottom:.2rem}
.fr input,.fr select,.fr textarea{width:100%;padding:.4rem .5rem;background:var(--bg);border:1px solid var(--bg3);color:var(--cream);font-family:var(--mono);font-size:.7rem}
.fr input:focus,.fr select:focus,.fr textarea:focus{outline:none;border-color:var(--leather)}
.row2{display:grid;grid-template-columns:1fr 1fr;gap:.5rem}
.acts{display:flex;gap:.4rem;justify-content:flex-end;margin-top:1rem}
.empty{text-align:center;padding:3rem;color:var(--cm);font-style:italic;font-size:.75rem}
@media(max-width:600px){.stats{grid-template-columns:repeat(2,1fr)}.row2{grid-template-columns:1fr}.toolbar{flex-direction:column}.search{min-width:100%}}
</style></head><body>
<div class="hdr"><h1><span>&#9670;</span> BILLFOLD</h1><button class="btn btn-p" onclick="openForm()">+ New Invoice</button></div>
<div class="main">
<div class="stats" id="stats"></div>
<div class="toolbar">
<input class="search" id="search" placeholder="Search clients, notes..." oninput="render()">
<div class="tabs" id="tabs"></div>
</div>
<div id="invoices"></div>
</div>
<div class="modal-bg" id="mbg" onclick="if(event.target===this)closeModal()"><div class="modal" id="mdl"></div></div>
<script>
var A='/api',invoices=[],filter='all',editId=null;

async function load(){var r=await fetch(A+'/invoices').then(function(r){return r.json()});invoices=r.invoices||[];renderStats();renderTabs();render();}

function renderStats(){
var total=0,paid=0,outstanding=0,count={all:invoices.length,draft:0,sent:0,paid:0,overdue:0};
invoices.forEach(function(i){
total+=i.amount;
if(i.status==='paid')paid+=i.amount;else outstanding+=i.amount;
count[i.status]=(count[i.status]||0)+1;
});
document.getElementById('stats').innerHTML=[
{l:'Total Invoiced',v:'$'+fmt(total/100),f:'all'},
{l:'Paid',v:'$'+fmt(paid/100),f:'paid',c:'var(--green)'},
{l:'Outstanding',v:'$'+fmt(outstanding/100),f:'outstanding',c:outstanding>0?'var(--gold)':'var(--cream)'},
{l:'Count',v:invoices.length,f:'all'}
].map(function(x){return '<div class="st'+(filter===x.f?' active':'')+'" onclick="setFilter(\''+x.f+'\')"><div class="st-v" style="'+(x.c?'color:'+x.c:'')+'">'+x.v+'</div><div class="st-l">'+x.l+'</div></div>'}).join('');
}

function renderTabs(){
document.getElementById('tabs').innerHTML=['all','draft','sent','paid','overdue'].map(function(t){
var c=invoices.filter(function(i){return t==='all'||i.status===t}).length;
return '<button class="tab'+(filter===t?' active':'')+'" onclick="setFilter(\''+t+'\')">'+t+' ('+c+')</button>';
}).join('');
}

function setFilter(f){filter=f;renderStats();renderTabs();render();}

function render(){
var q=(document.getElementById('search').value||'').toLowerCase();
var f=invoices;
if(filter!=='all'&&filter!=='outstanding')f=f.filter(function(i){return i.status===filter});
if(filter==='outstanding')f=f.filter(function(i){return i.status!=='paid'});
if(q)f=f.filter(function(i){return(i.name||'').toLowerCase().includes(q)||(i.notes||'').toLowerCase().includes(q)});
if(!f.length){document.getElementById('invoices').innerHTML='<div class="empty">No invoices found.</div>';return;}
var h='';f.forEach(function(i){
var overdue=i.status!=='paid'&&i.due_date&&new Date(i.due_date)<new Date();
var badge=overdue&&i.status!=='paid'?'overdue':i.status;
h+='<div class="inv"><div class="inv-top"><div class="inv-name">'+esc(i.name||'Untitled')+'</div>';
h+='<div style="display:flex;gap:.5rem;align-items:center"><span class="inv-amount">$'+fmt(i.amount/100)+'</span>';
h+='<span class="badge '+badge+'">'+badge+'</span></div></div>';
h+='<div class="inv-meta">';
if(i.due_date){var dd=i.due_date;h+='<span class="'+(overdue?'due-warn':'')+'">Due: '+dd+'</span>';}
h+='<span>Created: '+ft(i.created_at)+'</span>';
if(i.paid_at)h+='<span>Paid: '+ft(i.paid_at)+'</span>';
h+='</div>';
if(i.notes)h+='<div class="inv-notes">'+esc(i.notes)+'</div>';
h+='<div class="inv-actions">';
if(i.status==='draft')h+='<button class="btn btn-sm" onclick="setStatus(\''+i.id+'\',\'sent\')">Mark Sent</button>';
if(i.status==='sent'||overdue)h+='<button class="btn btn-sm btn-green" onclick="markPaid(\''+i.id+'\')">Mark Paid</button>';
if(i.status==='paid')h+='<button class="btn btn-sm" onclick="setStatus(\''+i.id+'\',\'draft\')">Reopen</button>';
h+='<button class="btn btn-sm" onclick="openEdit(\''+i.id+'\')">Edit</button>';
h+='<button class="btn btn-sm" onclick="del(\''+i.id+'\')" style="color:var(--red)">&#10005;</button>';
h+='</div></div>';
});
document.getElementById('invoices').innerHTML=h;
}

async function setStatus(id,status){await fetch(A+'/invoices/'+id,{method:'PUT',headers:{'Content-Type':'application/json'},body:JSON.stringify({status:status})});load();}
async function markPaid(id){await fetch(A+'/invoices/'+id,{method:'PUT',headers:{'Content-Type':'application/json'},body:JSON.stringify({status:'paid',paid_at:new Date().toISOString()})});load();}
async function del(id){if(!confirm('Delete this invoice?'))return;await fetch(A+'/invoices/'+id,{method:'DELETE'});load();}

function formHTML(inv){
var i=inv||{name:'',amount:0,due_date:'',status:'draft',notes:''};
var isEdit=!!inv;
var h='<h2>'+(isEdit?'EDIT INVOICE':'NEW INVOICE')+'</h2>';
h+='<div class="fr"><label>Client / Description *</label><input id="f-name" value="'+esc(i.name)+'" placeholder="e.g. Acme Corp - March retainer"></div>';
h+='<div class="row2"><div class="fr"><label>Amount ($)</label><input id="f-amount" type="number" step="0.01" value="'+(i.amount/100).toFixed(2)+'" placeholder="500.00"></div>';
h+='<div class="fr"><label>Due Date</label><input id="f-due" type="date" value="'+esc(i.due_date)+'"></div></div>';
if(isEdit){h+='<div class="fr"><label>Status</label><select id="f-status">';
['draft','sent','paid','overdue'].forEach(function(s){h+='<option value="'+s+'"'+(i.status===s?' selected':'')+'>'+s.charAt(0).toUpperCase()+s.slice(1)+'</option>';});
h+='</select></div>';}
h+='<div class="fr"><label>Notes</label><textarea id="f-notes" rows="3" placeholder="Additional details...">'+esc(i.notes)+'</textarea></div>';
h+='<div class="acts"><button class="btn" onclick="closeModal()">Cancel</button><button class="btn btn-p" onclick="submit()">'+(isEdit?'Save':'Create')+'</button></div>';
return h;
}

function openForm(){editId=null;document.getElementById('mdl').innerHTML=formHTML();document.getElementById('mbg').classList.add('open');document.getElementById('f-name').focus();}
function openEdit(id){var inv=null;for(var j=0;j<invoices.length;j++){if(invoices[j].id===id){inv=invoices[j];break;}}if(!inv)return;editId=id;document.getElementById('mdl').innerHTML=formHTML(inv);document.getElementById('mbg').classList.add('open');}
function closeModal(){document.getElementById('mbg').classList.remove('open');editId=null;}

async function submit(){
var name=document.getElementById('f-name').value.trim();
if(!name){alert('Client/description is required');return;}
var body={name:name,amount:Math.round(parseFloat(document.getElementById('f-amount').value||0)*100),due_date:document.getElementById('f-due').value,notes:document.getElementById('f-notes').value.trim()};
if(editId){var sel=document.getElementById('f-status');if(sel)body.status=sel.value;
await fetch(A+'/invoices/'+editId,{method:'PUT',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});}
else{await fetch(A+'/invoices',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});}
closeModal();load();
}

function fmt(n){return n.toLocaleString('en-US',{minimumFractionDigits:2,maximumFractionDigits:2});}
function ft(t){if(!t)return'';try{return new Date(t).toLocaleDateString('en-US',{month:'short',day:'numeric',year:'numeric'})}catch(e){return t;}}
function esc(s){if(!s)return'';var d=document.createElement('div');d.textContent=s;return d.innerHTML;}
document.addEventListener('keydown',function(e){if(e.key==='Escape')closeModal();});
load();
</script></body></html>`
