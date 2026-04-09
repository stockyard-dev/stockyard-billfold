package server

import "net/http"

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(dashHTML))
}

const dashHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1.0">
<title>Billfold</title>
<link href="https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500;700&display=swap" rel="stylesheet">
<style>
:root{--bg:#1a1410;--bg2:#241e18;--bg3:#2e261e;--rust:#e8753a;--leather:#a0845c;--cream:#f0e6d3;--cd:#bfb5a3;--cm:#7a7060;--gold:#d4a843;--green:#4a9e5c;--red:#c94444;--mono:'JetBrains Mono',monospace}
*{margin:0;padding:0;box-sizing:border-box}
body{background:var(--bg);color:var(--cream);font-family:var(--mono);line-height:1.5}
.hdr{padding:1rem 1.5rem;border-bottom:1px solid var(--bg3);display:flex;justify-content:space-between;align-items:center;gap:1rem;flex-wrap:wrap}
.hdr h1{font-size:.9rem;letter-spacing:2px}
.hdr h1 span{color:var(--rust)}
.main{padding:1.5rem;max-width:960px;margin:0 auto}
.stats{display:grid;grid-template-columns:repeat(auto-fit,minmax(140px,1fr));gap:.5rem;margin-bottom:1rem}
.st{background:var(--bg2);border:1px solid var(--bg3);padding:.7rem;text-align:center}
.st-v{font-size:1.3rem;font-weight:700;color:var(--gold)}
.st-l{font-size:.5rem;color:var(--cm);text-transform:uppercase;letter-spacing:1px;margin-top:.2rem}
.toolbar{display:flex;gap:.5rem;margin-bottom:1rem;flex-wrap:wrap;align-items:center}
.search{flex:1;min-width:180px;padding:.4rem .6rem;background:var(--bg2);border:1px solid var(--bg3);color:var(--cream);font-family:var(--mono);font-size:.7rem}
.search:focus{outline:none;border-color:var(--leather)}
.count-label{font-size:.6rem;color:var(--cm);margin-bottom:.5rem}
.item{background:var(--bg2);border:1px solid var(--bg3);padding:.8rem 1rem;margin-bottom:.5rem;transition:border-color .2s}
.item:hover{border-color:var(--leather)}
.item-top{display:flex;justify-content:space-between;align-items:flex-start;gap:.8rem}
.item-title{font-size:.85rem;font-weight:700;flex:1}
.item-meta{font-size:.55rem;color:var(--cm);margin-top:.3rem;display:flex;gap:.6rem;flex-wrap:wrap}
.item-meta-sep{color:var(--bg3)}
.item-actions{display:flex;gap:.3rem;flex-shrink:0;margin-left:.5rem}
.item-extra{font-size:.58rem;color:var(--cd);margin-top:.4rem;padding-top:.35rem;border-top:1px dashed var(--bg3);display:flex;flex-direction:column;gap:.15rem}
.item-extra-row{display:flex;gap:.4rem}
.item-extra-label{color:var(--cm);text-transform:uppercase;letter-spacing:.5px;min-width:90px}
.item-extra-val{color:var(--cream)}
.btn{font-size:.6rem;padding:.25rem .5rem;cursor:pointer;border:1px solid var(--bg3);background:var(--bg);color:var(--cd);transition:all .2s;font-family:var(--mono)}
.btn:hover{border-color:var(--leather);color:var(--cream)}
.btn-p{background:var(--rust);border-color:var(--rust);color:#fff}
.btn-sm{font-size:.55rem;padding:.2rem .4rem}
.modal-bg{display:none;position:fixed;inset:0;background:rgba(0,0,0,.65);z-index:100;align-items:center;justify-content:center}
.modal-bg.open{display:flex}
.modal{background:var(--bg2);border:1px solid var(--bg3);padding:1.5rem;width:480px;max-width:92vw;max-height:90vh;overflow-y:auto}
.modal h2{font-size:.8rem;margin-bottom:1rem;color:var(--rust);letter-spacing:1px}
.fr{margin-bottom:.6rem}
.fr label{display:block;font-size:.55rem;color:var(--cm);text-transform:uppercase;letter-spacing:1px;margin-bottom:.2rem}
.fr input,.fr select,.fr textarea{width:100%;padding:.4rem .5rem;background:var(--bg);border:1px solid var(--bg3);color:var(--cream);font-family:var(--mono);font-size:.7rem}
.fr input:focus,.fr select:focus,.fr textarea:focus{outline:none;border-color:var(--leather)}
.fr-checkbox{display:flex;align-items:center;gap:.5rem;margin-bottom:.6rem}
.fr-checkbox input{width:auto;margin:0}
.fr-checkbox label{display:inline;font-size:.65rem;color:var(--cd);text-transform:none;letter-spacing:0;margin:0}
.fr-section{margin-top:1rem;padding-top:.8rem;border-top:1px solid var(--bg3)}
.fr-section-label{font-size:.55rem;color:var(--rust);text-transform:uppercase;letter-spacing:1px;margin-bottom:.5rem}
.row2{display:grid;grid-template-columns:1fr 1fr;gap:.5rem}
.acts{display:flex;gap:.4rem;justify-content:flex-end;margin-top:1rem}
.empty{text-align:center;padding:3rem;color:var(--cm);font-style:italic;font-size:.85rem}
@media(max-width:600px){.row2{grid-template-columns:1fr}.toolbar{flex-direction:column;align-items:stretch}.search{min-width:100%}.trial-bar{flex-direction:column;align-items:stretch}.trial-bar input.key-input{width:100%}}
.trial-bar{display:none;background:linear-gradient(90deg,#3a2419,#2e1c14);border-bottom:2px solid var(--rust);padding:.7rem 1.5rem;font-family:var(--mono);font-size:.68rem;color:var(--cream);align-items:center;gap:1rem;flex-wrap:wrap}
.trial-bar.show{display:flex}
.trial-bar-msg{flex:1;min-width:240px;line-height:1.5}
.trial-bar-msg strong{color:var(--rust);text-transform:uppercase;letter-spacing:1px;font-size:.6rem;display:block;margin-bottom:.15rem}
.trial-bar-actions{display:flex;gap:.5rem;align-items:center;flex-wrap:wrap}
.trial-bar a.btn-trial{background:var(--rust);color:#fff;padding:.4rem .8rem;text-decoration:none;font-size:.65rem;text-transform:uppercase;letter-spacing:1px;font-weight:700;border:1px solid var(--rust);transition:all .2s}
.trial-bar a.btn-trial:hover{background:#f08545;border-color:#f08545}
.trial-bar-divider{color:var(--cm);font-size:.6rem}
.trial-bar input.key-input{padding:.4rem .5rem;background:var(--bg);border:1px solid var(--bg3);color:var(--cream);font-family:var(--mono);font-size:.6rem;width:200px}
.trial-bar input.key-input:focus{outline:none;border-color:var(--rust)}
.trial-bar button.btn-activate{padding:.4rem .7rem;background:var(--bg2);color:var(--cream);border:1px solid var(--leather);font-family:var(--mono);font-size:.6rem;cursor:pointer;text-transform:uppercase;letter-spacing:1px}
.trial-bar button.btn-activate:hover{background:var(--bg3)}
.trial-bar button.btn-activate:disabled{opacity:.5;cursor:wait}
.trial-msg{font-size:.6rem;color:var(--cm);margin-left:.5rem}
.trial-msg.error{color:#e74c3c}
.trial-msg.success{color:#4ade80}
.btn-disabled-trial{opacity:.45;cursor:not-allowed!important}
</style>
</head>
<body>

<div class="trial-bar" id="trial-bar">
<div class="trial-bar-msg">
<strong>Trial Required</strong>
You can view your existing invoices, but creating, editing, or deleting is locked until you start a 14-day free trial.
</div>
<div class="trial-bar-actions">
<a class="btn-trial" href="https://stockyard.dev/" target="_blank" rel="noopener">Start 14-Day Trial</a>
<span class="trial-bar-divider">or</span>
<input type="text" class="key-input" id="trial-key-input" placeholder="SY-..." autocomplete="off" spellcheck="false">
<button class="btn-activate" id="trial-activate-btn" onclick="activateLicense()">Activate</button>
<span class="trial-msg" id="trial-msg"></span>
</div>
</div>

<div class="hdr">
<h1 id="dash-title"><span>&#9670;</span> BILLFOLD</h1>
<button class="btn btn-p" onclick="openForm()">+ New</button>
</div>

<div class="main">
<div class="stats" id="stats"></div>
<div class="toolbar">
<input class="search" id="search" placeholder="Search..." oninput="render()">
</div>
<div class="count-label" id="count"></div>
<div id="list"></div>
</div>

<div class="modal-bg" id="mbg" onclick="if(event.target===this)closeModal()">
<div class="modal" id="mdl"></div>
</div>

<script>
var A='/api';
var RESOURCE='invoices';
var TITLE_FIELD='name';

var fields=[{"name": "id", "label": "Id", "type": "text"}, {"name": "name", "label": "Name", "type": "text"}, {"name": "amount", "label": "Amount", "type": "money"}, {"name": "due_date", "label": "Due Date", "type": "date"}, {"name": "status", "label": "Status", "type": "text"}, {"name": "line_items", "label": "Line Items", "type": "text"}, {"name": "notes", "label": "Notes", "type": "textarea"}, {"name": "paid_at", "label": "Paid At", "type": "text"}, {"name": "created_at", "label": "Created At", "type": "text"}];

var items=[],editId=null;

function fmtMoney(cents){
var n=parseInt(cents||0,10);
if(isNaN(n))return'$0.00';
var sign=n<0?'-':'';
n=Math.abs(n);
return sign+'$'+(n/100).toFixed(2);
}

function parseMoney(str){
if(!str)return 0;
var s=String(str).replace(/[^0-9.\-]/g,'');
if(!s)return 0;
var n=parseFloat(s);
if(isNaN(n))return 0;
return Math.round(n*100);
}

function fmtDate(s){
if(!s)return'';
try{
var d=new Date(s);
if(isNaN(d.getTime()))return s;
return d.toLocaleDateString('en-US',{year:'numeric',month:'short',day:'numeric'});
}catch(e){return s}
}

async function load(){
try{
var r=await fetch(A+'/'+RESOURCE).then(function(r){return r.json()});
var list=r[RESOURCE]||[];
try{
var extras=await fetch(A+'/extras/'+RESOURCE).then(function(r){return r.json()});
list.forEach(function(it){
var ex=extras[it.id];
if(!ex)return;
Object.keys(ex).forEach(function(k){if(it[k]===undefined)it[k]=ex[k]});
});
}catch(e){}
items=list;
}catch(e){
console.error('load failed',e);
items=[];
}
renderStats();
render();
}

function renderStats(){
var total=items.length;
document.getElementById('stats').innerHTML=
'<div class="st"><div class="st-v">'+total+'</div><div class="st-l">Total</div></div>';
}

function render(){
var q=(document.getElementById('search').value||'').toLowerCase();
var f=items;
if(q){
f=f.filter(function(i){
return Object.keys(i).some(function(k){
var v=i[k];
return v!==null&&v!==undefined&&String(v).toLowerCase().includes(q);
});
});
}
document.getElementById('count').textContent=f.length+' item'+(f.length!==1?'s':'');
if(!f.length){
var msg=window._emptyMsg||'No items yet. Click "+ New" to add one.';
document.getElementById('list').innerHTML='<div class="empty">'+esc(msg)+'</div>';
return;
}
var h='';
f.forEach(function(i){h+=itemHTML(i)});
document.getElementById('list').innerHTML=h;
}

function itemHTML(i){
var title=i[TITLE_FIELD]||'(untitled)';
var h='<div class="item"><div class="item-top">';
h+='<div class="item-title">'+esc(String(title))+'</div>';
if(!window._trialRequired){
h+='<div class="item-actions">';
h+='<button class="btn btn-sm" onclick="openEdit(\''+i.id+'\')">Edit</button>';
h+='<button class="btn btn-sm" onclick="del(\''+i.id+'\')" style="color:var(--red)">&#10005;</button>';
h+='</div>';
}
h+='</div>';

// Native field summary on meta line (skip the title field, skip empty)
var meta=[];
fields.forEach(function(f){
if(f.isCustom)return;
if(f.name===TITLE_FIELD||f.name==='id'||f.name==='created_at')return;
var v=i[f.name];
if(v===undefined||v===null||v===''||v===0)return;
var disp=String(v);
if(f.type==='money')disp=fmtMoney(v);
else if(f.type==='date'||f.type==='datetime')disp=fmtDate(v);
else if(disp.length>30)disp=disp.substring(0,30)+'…';
meta.push('<span><strong style="color:var(--cd)">'+esc(f.label)+':</strong> '+esc(disp)+'</span>');
});
if(i.created_at)meta.push('<span style="color:var(--cm)">'+esc(fmtDate(i.created_at))+'</span>');
if(meta.length){
h+='<div class="item-meta">'+meta.join('<span class="item-meta-sep">·</span>')+'</div>';
}

// Custom fields from personalization
var customRows='';
fields.forEach(function(f){
if(!f.isCustom)return;
var v=i[f.name];
if(v===undefined||v===null||v==='')return;
customRows+='<div class="item-extra-row">';
customRows+='<span class="item-extra-label">'+esc(f.label)+'</span>';
customRows+='<span class="item-extra-val">'+esc(String(v))+'</span>';
customRows+='</div>';
});
if(customRows)h+='<div class="item-extra">'+customRows+'</div>';

h+='</div>';
return h;
}

function fieldByName(n){
for(var i=0;i<fields.length;i++)if(fields[i].name===n)return fields[i];
return null;
}

function fieldHTML(f,value){
var v=value;
if(v===undefined||v===null)v='';
var req=f.required?' *':'';

if(f.type==='checkbox'){
return '<div class="fr-checkbox"><input type="checkbox" id="f-'+f.name+'"'+(v?' checked':'')+'><label for="f-'+f.name+'">'+esc(f.label)+'</label></div>';
}

var h='<div class="fr"><label>'+esc(f.label)+req+'</label>';

if(f.type==='select'){
h+='<select id="f-'+f.name+'">';
if(!f.required)h+='<option value="">Select...</option>';
(f.options||[]).forEach(function(o){
var sel=(String(v)===String(o))?' selected':'';
var disp=(typeof o==='string')?(o.charAt(0).toUpperCase()+o.slice(1)):String(o);
h+='<option value="'+esc(String(o))+'"'+sel+'>'+esc(disp)+'</option>';
});
h+='</select>';
}else if(f.type==='textarea'){
h+='<textarea id="f-'+f.name+'" rows="3">'+esc(String(v))+'</textarea>';
}else if(f.type==='money'){
var displayVal=v?fmtMoney(v).replace('$',''):'';
h+='<input type="text" id="f-'+f.name+'" value="'+esc(displayVal)+'" placeholder="0.00">';
}else if(f.type==='date'){
h+='<input type="date" id="f-'+f.name+'" value="'+esc(String(v).substring(0,10))+'">';
}else if(f.type==='datetime'){
h+='<input type="datetime-local" id="f-'+f.name+'" value="'+esc(String(v).substring(0,16))+'">';
}else if(f.type==='number'||f.type==='integer'){
h+='<input type="number" id="f-'+f.name+'" value="'+esc(String(v))+'">';
}else{
h+='<input type="text" id="f-'+f.name+'" value="'+esc(String(v))+'">';
}

h+='</div>';
return h;
}

function formHTML(item){
var i=item||{};
var isEdit=!!item;
var h='<h2>'+(isEdit?'EDIT':'NEW')+'</h2>';

// Native fields first
var nativeFields=fields.filter(function(f){
return !f.isCustom&&f.name!=='id'&&f.name!=='created_at';
});
nativeFields.forEach(function(f){
h+=fieldHTML(f,i[f.name]);
});

// Custom fields injected by personalization
var customFields=fields.filter(function(f){return f.isCustom});
if(customFields.length){
var sectionLabel=window._customSectionLabel||'Additional Details';
h+='<div class="fr-section"><div class="fr-section-label">'+esc(sectionLabel)+'</div>';
customFields.forEach(function(f){h+=fieldHTML(f,i[f.name])});
h+='</div>';
}

h+='<div class="acts">';
h+='<button class="btn" onclick="closeModal()">Cancel</button>';
h+='<button class="btn btn-p" onclick="submit()">'+(isEdit?'Save':'Create')+'</button>';
h+='</div>';
return h;
}

function openForm(){
editId=null;
document.getElementById('mdl').innerHTML=formHTML();
document.getElementById('mbg').classList.add('open');
}

function openEdit(id){
var x=null;
for(var j=0;j<items.length;j++){if(items[j].id===id){x=items[j];break}}
if(!x)return;
editId=id;
document.getElementById('mdl').innerHTML=formHTML(x);
document.getElementById('mbg').classList.add('open');
}

function closeModal(){
document.getElementById('mbg').classList.remove('open');
editId=null;
}

async function submit(){
var body={};
var extras={};
fields.forEach(function(f){
if(f.name==='id'||f.name==='created_at')return;
var el=document.getElementById('f-'+f.name);
if(!el)return;
var val;
if(f.type==='checkbox')val=el.checked?1:0;
else if(f.type==='money')val=parseMoney(el.value);
else if(f.type==='number')val=parseFloat(el.value)||0;
else if(f.type==='integer')val=parseInt(el.value,10)||0;
else val=el.value;
if(f.isCustom)extras[f.name]=val;
else body[f.name]=val;
});

var savedId=editId;
try{
if(editId){
var r1=await fetch(A+'/'+RESOURCE+'/'+editId,{method:'PUT',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});
if(!r1.ok){var e1=await r1.json().catch(function(){return{}});alert(e1.error||'Save failed');return}
}else{
var r2=await fetch(A+'/'+RESOURCE,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});
if(!r2.ok){var e2=await r2.json().catch(function(){return{}});alert(e2.error||'Save failed');return}
var created=await r2.json();
savedId=created.id;
}
if(savedId&&Object.keys(extras).length){
await fetch(A+'/extras/'+RESOURCE+'/'+savedId,{method:'PUT',headers:{'Content-Type':'application/json'},body:JSON.stringify(extras)}).catch(function(){});
}
}catch(e){
alert('Network error: '+e.message);
return;
}

closeModal();
load();
}

async function del(id){
if(!confirm('Delete this item?'))return;
await fetch(A+'/'+RESOURCE+'/'+id,{method:'DELETE'});
load();
}

function esc(s){
if(s===undefined||s===null)return'';
var d=document.createElement('div');
d.textContent=String(s);
return d.innerHTML;
}

document.addEventListener('keydown',function(e){if(e.key==='Escape')closeModal()});

(function loadPersonalization(){
fetch('/api/config').then(function(r){return r.json()}).then(function(cfg){
if(!cfg||typeof cfg!=='object')return;

if(cfg.dashboard_title){
var h1=document.getElementById('dash-title');
if(h1)h1.innerHTML='<span>&#9670;</span> '+esc(cfg.dashboard_title);
document.title=cfg.dashboard_title;
}

if(cfg.empty_state_message)window._emptyMsg=cfg.empty_state_message;
if(cfg.primary_label)window._customSectionLabel=cfg.primary_label+' Details';

if(Array.isArray(cfg.custom_fields)){
cfg.custom_fields.forEach(function(cf){
if(!cf||!cf.name||!cf.label)return;
if(fieldByName(cf.name))return;
fields.push({
name:cf.name,
label:cf.label,
type:cf.type||'text',
options:cf.options||[],
isCustom:true
});
});
}
}).catch(function(){
}).finally(function(){
checkTrialState();
load();
});
})();

// ─── trial-required license gating ───
// window._trialRequired is checked by render() / itemHTML() to decide
// whether to emit Edit and Delete buttons on each card.
window._trialRequired=false;

async function checkTrialState(){
try{
var resp=await fetch(A+'/tier');
if(!resp.ok)return;
var data=await resp.json();
window._trialRequired=!!data.trial_required;
if(window._trialRequired){
document.getElementById('trial-bar').classList.add('show');
disableWriteControls();
if(typeof render==='function')render();
}else{
document.getElementById('trial-bar').classList.remove('show');
}
}catch(e){}
}

function disableWriteControls(){
var buttons=document.querySelectorAll('.hdr .btn, .hdr .btn-p');
buttons.forEach(function(b){
var t=b.textContent||'';
if(t.indexOf('New')!==-1||t.indexOf('Add')!==-1||t.indexOf('Import')!==-1){
b.classList.add('btn-disabled-trial');
b.title='Locked: trial required';
b.onclick=function(e){
e.preventDefault();
showTrialNudge();
return false;
};
}
});
}

function showTrialNudge(){
var input=document.getElementById('trial-key-input');
if(input){
input.focus();
input.style.borderColor='var(--rust)';
setTimeout(function(){if(input)input.style.borderColor=''},1500);
}
}

async function activateLicense(){
var input=document.getElementById('trial-key-input');
var btn=document.getElementById('trial-activate-btn');
var msg=document.getElementById('trial-msg');
if(!input||!btn||!msg)return;
var key=(input.value||'').trim();
if(!key){
msg.className='trial-msg error';
msg.textContent='Paste your license key first';
input.focus();
return;
}
btn.disabled=true;
msg.className='trial-msg';
msg.textContent='Activating...';
try{
var resp=await fetch(A+'/license/activate',{
method:'POST',
headers:{'Content-Type':'application/json'},
body:JSON.stringify({license_key:key})
});
var data=await resp.json();
if(!resp.ok){
msg.className='trial-msg error';
msg.textContent=data.error||'Activation failed';
btn.disabled=false;
return;
}
msg.className='trial-msg success';
msg.textContent='Activated. Reloading...';
setTimeout(function(){location.reload()},800);
}catch(e){
msg.className='trial-msg error';
msg.textContent='Network error: '+e.message;
btn.disabled=false;
}
}

document.addEventListener('DOMContentLoaded',function(){
var input=document.getElementById('trial-key-input');
if(input){
input.addEventListener('keydown',function(e){
if(e.key==='Enter')activateLicense();
});
}
});
</script>
</body>
</html>` + ""
