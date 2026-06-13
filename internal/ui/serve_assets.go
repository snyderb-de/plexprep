package ui

// convertOverlays are the serve-mode-only modal (convert options) and the live
// status dashboard. Both ship hidden; serveJS toggles them.
const convertOverlays = `
<div id="opts" class="modal" hidden><div class="modalbox">
 <div class="modalhead">&#9484;&#9472; CONVERT OPTIONS &#9472;&#9488;</div>
 <div class="optrow"><span class="optk">profile</span><div class="optv">
   <label><input type="radio" name="prof" value="" checked> <b>zero-transcode</b> &middot; x264 CRF18 for legacy, copy modern (SD/HD)</label>
   <label><input type="radio" name="prof" value="4k"> <b>4K UHD</b> &middot; x265/HEVC CRF20 for legacy, keep HEVC</label>
   <label><input type="radio" name="prof" value="audio"> <b>audio-only</b> &middot; copy video, add AAC stereo only</label>
   <label><input type="radio" name="prof" value="shrink"> <b>shrink</b> &middot; re-encode every file at a custom CRF to cut size</label>
 </div></div>
 <div class="optrow" id="opt-crfrow" hidden><span class="optk">quality</span><div class="optv">
   <input type="range" id="opt-crf" min="18" max="30" step="1" value="20">
   <div class="crflabel">CRF <b id="opt-crf-val">20</b> &mdash; <span id="opt-crf-tag">visually lossless</span></div>
 </div></div>
 <div class="optrow"><span class="optk">output</span><div class="optv">
   <label><input type="checkbox" id="opt-replace"> <b>replace in place</b> &middot; output takes source name, source &rarr; <span class="amber">.original</span> backup</label>
   <label><input type="checkbox" id="opt-delete"> <b>delete original</b> after convert <span class="red">(irreversible &mdash; frees space mid-batch)</span></label>
 </div></div>
 <div class="optsummary" id="opt-summary"></div>
 <div class="optbtns"><button id="opt-start" class="selbtn cta">start convert</button><button id="opt-cancel" class="selbtn">cancel</button></div>
</div></div>

<div id="status" class="status" hidden><div class="statwrap">
 <div class="bar"><span class="dot r"></span><span class="dot y"></span><span class="dot g"></span>
   <span class="bartitle">plexprep // <span id="st-phase">CONVERTING</span></span><span class="statclock" id="st-clock">0s</span></div>
 <div class="statbody">
   <pre class="statgrid" id="st-grid"></pre>
   <div class="statnow"><div class="statnowname" id="st-name">&mdash;</div>
     <div class="meterbig"><span id="st-pbar" class="meterfill"></span></div>
     <div class="statnowmeta"><span id="st-pct">0%</span> <span id="st-speed" class="dimx"></span> <span id="st-eta" class="amber"></span></div></div>
   <div class="statoverall"><div class="statlbl">OVERALL <span id="st-ocount" class="dimx"></span></div>
     <div class="meterbig"><span id="st-obar" class="meterfill ofill"></span></div></div>
   <div class="statlog" id="st-log"></div>
   <div class="statbtns"><button id="st-close" class="selbtn cta" hidden>done &mdash; close</button>
     <button id="st-abort" class="selbtn">abort after current</button>
     <button id="st-abort-now" class="selbtn red">abort now</button></div>
 </div>
</div></div>`

// serveCSS styles the serve-only overlays (modal + status dashboard).
const serveCSS = `<style>
.selbtn.cta{color:var(--bg);background:var(--amber);border-color:var(--amber);font-weight:700}
.selbtn.cta:hover{background:var(--bright);border-color:var(--bright);color:var(--bg)}
.selbtn.red{color:var(--red);border-color:var(--red)}
.selbtn.red:hover{background:var(--red);color:var(--bg)}
.amber{color:var(--amber)}.red{color:var(--red)}.dimx{color:var(--dim)}
.modal{position:fixed;inset:0;z-index:40;display:none;align-items:center;justify-content:center;
  background:#000a;backdrop-filter:blur(2px);padding:20px}
.modal:not([hidden]){display:flex}
.modalbox{background:var(--panel);border:1px solid var(--fg);border-radius:6px;max-width:640px;width:100%;
  padding:18px 20px;box-shadow:0 20px 60px -20px #000,0 0 0 1px #0a1a10}
.modalhead{color:var(--fg);font-weight:700;letter-spacing:1px;margin-bottom:14px}
.optrow{display:flex;gap:14px;margin:10px 0;align-items:flex-start}
.optk{color:var(--amber);text-transform:uppercase;font-size:11px;letter-spacing:.5px;width:62px;flex:none;padding-top:3px}
.optv{display:flex;flex-direction:column;gap:7px;font-size:12.5px;color:var(--mid)}
.optv label{cursor:pointer}.optv b{color:var(--bright)}
.optv input{accent-color:var(--fg);vertical-align:middle;margin-right:6px}
#opt-crf{-webkit-appearance:none;appearance:none;width:100%;height:6px;border-radius:3px;
  background:linear-gradient(to right,#3ecf6e,#e8c547,#e0524a);outline:none}
#opt-crf::-webkit-slider-thumb{-webkit-appearance:none;width:14px;height:14px;border-radius:50%;
  background:var(--bright);border:1px solid var(--fg);cursor:pointer}
#opt-crf::-moz-range-thumb{width:14px;height:14px;border-radius:50%;
  background:var(--bright);border:1px solid var(--fg);cursor:pointer}
.crflabel{margin-top:6px;font-size:12px;color:var(--mid)}
.optsummary{margin:14px 0 6px;color:var(--bright);font-size:12.5px;border-top:1px solid var(--line);padding-top:12px}
.optbtns{display:flex;gap:10px;margin-top:12px}
/* status dashboard */
.status{position:fixed;inset:0;z-index:50;background:var(--bg);overflow:auto;padding:clamp(14px,3vw,40px)}
.statwrap{max-width:1100px;margin:0 auto;border:1px solid var(--line);border-radius:6px;background:var(--panel);
  box-shadow:0 0 0 1px #0a1a10,0 18px 60px -30px #000}
.statclock{margin-left:auto;color:var(--amber);font-variant-numeric:tabular-nums}
.statbody{padding:18px 20px 22px}
.statgrid{margin:0 0 16px;color:var(--fg);font-size:13px;line-height:1.6;white-space:pre-wrap}
.statnowname{color:var(--bright);font-weight:700;margin-bottom:6px;word-break:break-all}
.meterbig{height:16px;background:#0a0d0a;border:1px solid var(--line);border-radius:3px;overflow:hidden}
.meterfill{display:block;height:100%;width:0;background:var(--amber);transition:width .2s linear}
.meterfill.ofill{background:var(--fg)}
.statnowmeta{margin-top:5px;font-size:12px;color:var(--mid);display:flex;gap:14px}
.statnowmeta #st-pct{color:var(--bright);font-weight:700}
.statoverall{margin-top:16px}
.statlbl{font-size:11px;letter-spacing:.5px;color:var(--mid);margin-bottom:5px}
.statlog{margin-top:16px;height:240px;overflow:auto;background:#080a08;border:1px solid var(--line);
  border-radius:4px;padding:8px 10px;font-size:11.5px;line-height:1.5}
.statlog div{white-space:pre-wrap;word-break:break-all}
.statlog .ok{color:var(--fg)}.statlog .warn{color:var(--amber)}.statlog .err{color:var(--red)}.statlog .dimx{color:var(--dim)}
.statbtns{margin-top:16px;display:flex;gap:10px}
</style>`

// crfControlJS is the shared CRF-slider behaviour for both UIs: continuous
// green→red gradient on the value/quality tag, show the quality row only for the
// shrink profile, and re-render the convert summary when the profile changes.
// It relies on $, picked, hb and renderSummary being in the enclosing scope.
const crfControlJS = `
 function crfColor(t){ var r=Math.round(62+(224-62)*t), g=Math.round(207+(82-207)*t), b=Math.round(110+(74-110)*t); return 'rgb('+r+','+g+','+b+')'; }
 function updateCRF(){
   var el=$('opt-crf'); if(!el) return;
   var v=+el.value, t=(v-18)/12, c=crfColor(t);
   var val=$('opt-crf-val'), tag=$('opt-crf-tag');
   val.textContent=v; val.style.color=c; tag.style.color=c;
   tag.textContent = t<0.17?'visually lossless': t<0.42?'good': t<0.75?'noticeable loss':'dogwater';
 }
 function syncProfRow(){ var prof=(document.querySelector('input[name=prof]:checked')||{}).value; $('opt-crfrow').hidden = prof!=='shrink'; renderSummary(); }
 var crfEl=$('opt-crf'); if(crfEl) crfEl.addEventListener('input', updateCRF);
 document.querySelectorAll('input[name=prof]').forEach(function(r){ r.addEventListener('change', syncProfRow); });
 updateCRF(); syncProfRow();`

// serveJS drives the convert flow: open options, POST the selection, stream
// newline-JSON progress into the status dashboard.
const serveJS = `<script>
(function(){
 var root=(document.querySelector('meta[name="pp-root"]')||{}).content||'';
 function $(id){return document.getElementById(id);}
 function hb(b){b=Math.round(b); if(Math.abs(b)<1024) return b+' B'; var u=['K','M','G','T','P','E'],i=-1,n=Math.abs(b); while(n>=1024&&i<u.length-1){n/=1024;i++;} return (b<0?'-':'')+n.toFixed(2)+' '+u[i]+'B';}
 function hd(s){s=Math.round(s); if(s<=0)return '0s'; var h=(s/3600)|0,m=((s%3600)/60)|0,x=s%60; if(h>0)return h+'h '+m+'m'; if(m>0)return m+'m '+x+'s'; return x+'s';}
 function picked(){var o=[]; document.querySelectorAll('.pick-cb').forEach(function(c){ if(c.checked) o.push({path:c.getAttribute('data-path'),size:+c.getAttribute('data-size')||0,saved:+c.getAttribute('data-saved')||0}); }); return o;}

 function renderSummary(){
   var p=picked(); if(!p.length) return;
   var sz=p.reduce(function(a,x){return a+x.size;},0), sv=p.reduce(function(a,x){return a+x.saved;},0);
   var prof=(document.querySelector('input[name=prof]:checked')||{}).value||'';
   var tail = prof==='shrink'
     ? ' <span class="dimx">&middot; savings depend on CRF (shown live during convert)</span>'
     : ' &rarr; reclaim '+(sv>=0?'<b class="save">'+hb(sv)+'</b>':'<b class="grow">&#9650; +'+hb(-sv)+' larger</b>')+' <span class="dimx">(estimate)</span>';
   $('opt-summary').innerHTML='<b>'+p.length+'</b> file'+(p.length===1?'':'s')+' &middot; '+hb(sz)+tail;
 }
` + crfControlJS + `

 var cv=$('sel-convert');
 if(cv) cv.addEventListener('click', function(){
   var p=picked(); if(!p.length) return;
   renderSummary();
   $('opts').hidden=false;
 });
 var oc=$('opt-cancel'); if(oc) oc.addEventListener('click', function(){ $('opts').hidden=true; });

 var clockT=null, t0=0, abort=false;
 function startClock(){ t0=Date.now(); clockT=setInterval(function(){ $('st-clock').textContent=hd((Date.now()-t0)/1000); },1000); }
 function stopClock(){ if(clockT) clearInterval(clockT); }
 function logln(cls,txt){ var d=document.createElement('div'); d.className=cls; d.textContent=txt; var L=$('st-log'); L.appendChild(d); L.scrollTop=L.scrollHeight; }

 function grid(s){
   $('st-grid').textContent =
     'files     : '+s.done+' / '+s.total+'   ('+s.ok+' ok · '+s.fail+' fail · '+s.skip+' skip)\n'+
     'reclaimed : '+hb(s.reclaim)+'\n'+
     'elapsed   : '+hd((Date.now()-t0)/1000)+(s.eta?('   eta ~'+hd(s.eta)):'');
 }

 var os_=$('opt-start');
 if(os_) os_.addEventListener('click', function(){
   var p=picked(); if(!p.length) return;
   var prof=(document.querySelector('input[name=prof]:checked')||{}).value||'';
   var crf=+(($('opt-crf')||{}).value||20);
   var body={root:root, profile:prof, crf:crf, replace:$('opt-replace').checked, delete:$('opt-delete').checked, paths:p.map(function(x){return x.path;})};
   $('opts').hidden=true; $('status').hidden=false; $('st-close').hidden=true; $('st-abort').hidden=false; $('st-abort-now').hidden=false; abort=false;
   $('st-log').innerHTML=''; $('st-phase').textContent='CONVERTING'; startClock();
   var s={total:p.length,done:0,ok:0,fail:0,skip:0,reclaim:0,eta:0}; grid(s);
   logln('dimx','POST /api/convert · '+p.length+' files · profile='+(prof||'zero')+(prof==='shrink'?' · crf='+crf:'')+(body.replace?' · replace':'')+(body.delete?' · delete':''));

   fetch('/api/convert',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)})
   .then(function(r){
     if(!r.ok||!r.body){ throw new Error('HTTP '+r.status); }
     var rd=r.body.getReader(), dec=new TextDecoder(), buf='';
     function pump(){ return rd.read().then(function(res){
       if(res.done){ return; }
       buf+=dec.decode(res.value,{stream:true});
       var lines=buf.split('\n'); buf=lines.pop();
       lines.forEach(function(ln){ if(ln.trim()) handle(JSON.parse(ln), s); });
       return pump();
     }); }
     return pump();
   })
   .then(function(){ finish(s); })
   .catch(function(e){ logln('err','stream error: '+e.message); finish(s); });
 });

 function handle(ev,s){
   if(ev.t==='start'){ logln('dimx','▸ ['+ev.idx+'/'+ev.total+'] '+ev.name); $('st-name').textContent='['+ev.idx+'/'+ev.total+'] '+ev.name; $('st-pbar').style.width='0%'; $('st-pct').textContent='0%'; $('st-speed').textContent=''; $('st-eta').textContent=''; }
   else if(ev.t==='progress'){ var p=Math.round(ev.frac*100); $('st-pbar').style.width=p+'%'; $('st-pct').textContent=p+'%'; $('st-speed').textContent=ev.speed?('@ '+ev.speed):''; if(ev.eta) $('st-eta').textContent='eta ~'+hd(ev.eta);
     var ov=(s.done+ev.frac)/s.total; $('st-obar').style.width=Math.round(ov*100)+'%'; $('st-ocount').textContent=s.done+'/'+s.total; }
   else if(ev.t==='done-file'){ s.done++; s.ok++; s.reclaim+=ev.saved; logln('ok','✓ '+ev.name+'  '+hb(ev.orig)+' → '+hb(ev.nu)+'  ('+(ev.saved>=0?hb(ev.saved)+' saved':'▲ +'+hb(-ev.saved)+' larger')+')'); $('st-obar').style.width=Math.round(s.done/s.total*100)+'%'; grid(s); }
   else if(ev.t==='skip'){ s.done++; s.skip++; logln('dimx','● '+ev.name+'  '+(ev.reason||'already optimal')); grid(s); }
   else if(ev.t==='fail'){ s.done++; s.fail++; logln('err','✘ '+ev.name+'  '+ev.err); grid(s); }
   else if(ev.t==='summary'){ s.reclaim=ev.reclaim; s.ok=ev.ok; s.fail=ev.fail; s.skip=ev.skip; grid(s); }
 }
 function finish(s){ stopClock(); $('st-phase').textContent='DONE'; $('st-obar').style.width='100%'; $('st-abort').hidden=true; $('st-abort-now').hidden=true; $('st-close').hidden=false; logln('ok','— complete · '+s.ok+' converted · reclaimed '+hb(s.reclaim)); }

 var sc=$('st-close'); if(sc) sc.addEventListener('click', function(){ location.reload(); });
 var sa=$('st-abort'); if(sa) sa.addEventListener('click', function(){ abort=true; sa.textContent='aborting…'; fetch('/api/abort',{method:'POST'}); });
 var san=$('st-abort-now'); if(san) san.addEventListener('click', function(){ abort=true; san.textContent='aborting…'; fetch('/api/abort-now',{method:'POST'}); });
})();
</script>`

// scanShell loads instantly and streams scan progress, then swaps to the
// finished report — the page never blocks on a whole-tree probe.
const scanShell = `<!doctype html><html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1"><title>plexprep // scanning</title>` +
	reportCSS + `<style>
.scwrap{max-width:760px}
.scname{color:var(--mid);font-size:12px;margin-top:8px;height:1.4em;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.meterbig{height:18px;background:#0a0d0a;border:1px solid var(--line);border-radius:3px;overflow:hidden;margin-top:10px}
.meterfill{display:block;height:100%;width:0;background:var(--fg);transition:width .15s linear}
.scnum{color:var(--bright);font-weight:700}
.scerr{color:var(--red);margin-top:12px}
</style></head><body><div class="crt"></div><main><div class="term">
<div class="bar"><span class="dot r"></span><span class="dot y"></span><span class="dot g"></span><span class="bartitle">plexprep — scanning</span></div>
<div class="body scwrap">
 <div class="prompt"><span class="usr">bag@plexprep</span>:<span class="pwd">~</span>$ plexprep --scan <span id="sc-path"></span><span class="cur"></span></div>
 <div class="done">probing media metadata <span class="dimx">(no full-file reads)</span></div>
 <div class="scnum"><span id="sc-done">0</span> / <span id="sc-total">?</span> files</div>
 <div class="meterbig"><span id="sc-bar" class="meterfill"></span></div>
 <div class="scname" id="sc-name">&nbsp;</div>
 <div class="scerr" id="sc-err" hidden></div>
</div></div></main>
<script>
(function(){
 function $(id){return document.getElementById(id);}
 var q=new URLSearchParams(location.search);
 var path=q.get('path')||'', rec=q.get('recursive')||'1';
 $('sc-path').textContent=path;
 var total=0;
 fetch('/api/scan?recursive='+rec+'&path='+encodeURIComponent(path)).then(function(r){
   if(!r.ok||!r.body) throw new Error('HTTP '+r.status);
   var rd=r.body.getReader(), dec=new TextDecoder(), buf='';
   function pump(){ return rd.read().then(function(res){
     if(res.done) return;
     buf+=dec.decode(res.value,{stream:true});
     var lines=buf.split('\n'); buf=lines.pop();
     lines.forEach(function(ln){ if(ln.trim()) handle(JSON.parse(ln)); });
     return pump();
   }); }
   return pump();
 }).catch(function(e){ err(e.message); });
 function handle(ev){
   if(ev.t==='begin'){ total=ev.total; $('sc-total').textContent=total||'?'; }
   else if(ev.t==='probe'){ $('sc-done').textContent=ev.done; if(ev.total) $('sc-bar').style.width=Math.round(ev.done/ev.total*100)+'%'; $('sc-name').textContent=ev.name||''; }
   else if(ev.t==='error'){ err(ev.msg); }
   else if(ev.t==='done'){ $('sc-bar').style.width='100%'; location.replace('/view?id='+ev.id); }
 }
 function err(m){ var e=$('sc-err'); e.hidden=false; e.innerHTML='scan error: '+m+'<br><br><a style="color:var(--fg)" href="/">&larr; back</a>'; }
})();
</script></body></html>`

// embedJS is the desktop (Wails) variant of serveJS: same options modal +
// status dashboard, but conversion runs via the bound Go App and progress
// arrives over Wails events (the report renders inside an iframe, so it reaches
// the runtime through window.parent).
const embedJS = `<script>
(function(){
 var P=window.parent||window;
 var RT=P.runtime||window.runtime;
 var APP=(P.go&&P.go.main&&P.go.main.App)||(window.go&&window.go.main&&window.go.main.App);
 function $(id){return document.getElementById(id);}
 function hb(b){b=Math.round(b); if(Math.abs(b)<1024) return b+' B'; var u=['K','M','G','T','P','E'],i=-1,n=Math.abs(b); while(n>=1024&&i<u.length-1){n/=1024;i++;} return (b<0?'-':'')+n.toFixed(2)+' '+u[i]+'B';}
 function hd(s){s=Math.round(s); if(s<=0)return '0s'; var h=(s/3600)|0,m=((s%3600)/60)|0,x=s%60; if(h>0)return h+'h '+m+'m'; if(m>0)return m+'m '+x+'s'; return x+'s';}
 function picked(){var o=[]; document.querySelectorAll('.pick-cb').forEach(function(c){ if(c.checked) o.push({path:c.getAttribute('data-path'),size:+c.getAttribute('data-size')||0,saved:+c.getAttribute('data-saved')||0}); }); return o;}

 function renderSummary(){
   var p=picked(); if(!p.length) return;
   var sz=p.reduce(function(a,x){return a+x.size;},0), sv=p.reduce(function(a,x){return a+x.saved;},0);
   var prof=(document.querySelector('input[name=prof]:checked')||{}).value||'';
   var tail = prof==='shrink'
     ? ' <span class="dimx">&middot; savings depend on CRF (shown live during convert)</span>'
     : ' &rarr; reclaim '+(sv>=0?'<b class="save">'+hb(sv)+'</b>':'<b class="grow">&#9650; +'+hb(-sv)+' larger</b>')+' <span class="dimx">(estimate)</span>';
   $('opt-summary').innerHTML='<b>'+p.length+'</b> file'+(p.length===1?'':'s')+' &middot; '+hb(sz)+tail;
 }
` + crfControlJS + `

 var cv=$('sel-convert');
 if(cv) cv.addEventListener('click', function(){
   var p=picked(); if(!p.length) return;
   renderSummary();
   $('opts').hidden=false;
 });
 var oc=$('opt-cancel'); if(oc) oc.addEventListener('click', function(){ $('opts').hidden=true; });

 var clockT=null,t0=0,s={total:0,done:0,ok:0,fail:0,skip:0,reclaim:0,eta:0};
 function startClock(){ t0=Date.now(); clockT=setInterval(function(){ $('st-clock').textContent=hd((Date.now()-t0)/1000); },1000); }
 function stopClock(){ if(clockT) clearInterval(clockT); }
 function logln(cls,txt){ var d=document.createElement('div'); d.className=cls; d.textContent=txt; var L=$('st-log'); L.appendChild(d); L.scrollTop=L.scrollHeight; }
 function grid(){ $('st-grid').textContent='files     : '+s.done+' / '+s.total+'   ('+s.ok+' ok · '+s.fail+' fail · '+s.skip+' skip)\nreclaimed : '+hb(s.reclaim)+'\nelapsed   : '+hd((Date.now()-t0)/1000); }
 function handle(ev){
   if(ev.t==='start'){ logln('dimx','▸ ['+ev.idx+'/'+ev.total+'] '+ev.name); $('st-name').textContent='['+ev.idx+'/'+ev.total+'] '+ev.name; $('st-pbar').style.width='0%'; $('st-pct').textContent='0%'; $('st-speed').textContent=''; $('st-eta').textContent=''; }
   else if(ev.t==='progress'){ var p=Math.round(ev.frac*100); $('st-pbar').style.width=p+'%'; $('st-pct').textContent=p+'%'; $('st-speed').textContent=ev.speed?('@ '+ev.speed):''; if(ev.eta) $('st-eta').textContent='eta ~'+hd(ev.eta); var ov=(s.done+ev.frac)/Math.max(1,s.total); $('st-obar').style.width=Math.round(ov*100)+'%'; $('st-ocount').textContent=s.done+'/'+s.total; }
   else if(ev.t==='done-file'){ s.done++; s.ok++; s.reclaim+=ev.saved; logln('ok','✓ '+ev.name+'  '+hb(ev.orig)+' → '+hb(ev.nu)+'  ('+(ev.saved>=0?hb(ev.saved)+' saved':'▲ +'+hb(-ev.saved)+' larger')+')'); $('st-obar').style.width=Math.round(s.done/Math.max(1,s.total)*100)+'%'; grid(); }
   else if(ev.t==='skip'){ s.done++; s.skip++; logln('dimx','● '+ev.name+'  '+(ev.reason||'already optimal')); grid(); }
   else if(ev.t==='fail'){ s.done++; s.fail++; logln('err','✘ '+ev.name+'  '+ev.err); grid(); }
   else if(ev.t==='summary'){ s.ok=ev.ok; s.fail=ev.fail; s.skip=ev.skip; s.reclaim=ev.reclaim; grid(); finish(); }
 }
 function finish(){ stopClock(); $('st-phase').textContent='DONE'; $('st-obar').style.width='100%'; $('st-abort').hidden=true; $('st-abort-now').hidden=true; $('st-close').hidden=false; logln('ok','— complete · '+s.ok+' converted · reclaimed '+hb(s.reclaim)); }

 if(RT&&RT.EventsOn) RT.EventsOn('pp:convert', handle);
 var os_=$('opt-start');
 if(os_) os_.addEventListener('click', function(){
   var p=picked(); if(!p.length) return;
   if(!APP){ alert('desktop bridge unavailable'); return; }
   var prof=(document.querySelector('input[name=prof]:checked')||{}).value||'';
   var crf=+(($('opt-crf')||{}).value||20);
   $('opts').hidden=true; $('status').hidden=false; $('st-close').hidden=true; $('st-abort').hidden=false; $('st-abort-now').hidden=false;
   $('st-log').innerHTML=''; $('st-phase').textContent='CONVERTING';
   s={total:p.length,done:0,ok:0,fail:0,skip:0,reclaim:0,eta:0}; startClock(); grid();
   logln('dimx','convert '+p.length+' files · profile='+(prof||'zero')+(prof==='shrink'?' · crf='+crf:'')+($('opt-replace').checked?' · replace':'')+($('opt-delete').checked?' · delete':''));
   APP.Convert(p.map(function(x){return x.path;}), prof, $('opt-replace').checked, $('opt-delete').checked, crf);
 });
 var sc=$('st-close'); if(sc) sc.addEventListener('click', function(){ if(P!==window&&P.ppDone){ P.ppDone(); } else { location.reload(); } });
 var sa=$('st-abort'); if(sa) sa.addEventListener('click', function(){ if(APP&&APP.Abort) APP.Abort(); sa.textContent='aborting…'; });
 var san=$('st-abort-now'); if(san) san.addEventListener('click', function(){ if(APP&&APP.AbortNow) APP.AbortNow(); san.textContent='aborting…'; });
})();
</script>`

// pickerHTML is the serve-mode landing page: choose a folder/file, toggle
// recursion, then scan. It uses /api/ls for a server-side directory browser.
const pickerHTML = `<!doctype html><html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1"><title>plexprep // pick</title>` +
	reportCSS + `<style>
.pickwrap{max-width:900px}
.pkrow{display:flex;gap:8px;margin:14px 0;align-items:center;flex-wrap:wrap}
.pkin{flex:1;min-width:280px;font:inherit;font-size:13px;color:var(--bright);background:#080a08;
  border:1px solid var(--line);border-radius:3px;padding:7px 10px}
.pkin:focus{outline:none;border-color:var(--fg)}
.crumbs{color:var(--mid);font-size:12px;margin:8px 0;word-break:break-all}
.crumbs a{color:var(--amber);text-decoration:none}.crumbs a:hover{text-decoration:underline}
.lsbox{border:1px solid var(--line);border-radius:4px;background:#080a08;max-height:46vh;overflow:auto;margin-top:8px}
.lsitem{display:flex;justify-content:space-between;gap:10px;padding:5px 12px;cursor:pointer;border-bottom:1px solid #0d1f12}
.lsitem:hover{background:#0f1f14}.lsitem .nm{color:var(--bright)}.lsitem.dir .nm:before{content:"▸ ";color:var(--amber)}
.lsitem.file .nm:before{content:"  ";}.lsitem .mt{color:var(--dim);font-size:11px}
.opt2{color:var(--mid);font-size:12.5px}.opt2 input{accent-color:var(--fg);vertical-align:middle;margin-right:6px}
</style></head><body><div class="crt"></div><main><div class="term">
<div class="bar"><span class="dot r"></span><span class="dot y"></span><span class="dot g"></span><span class="bartitle">plexprep — pick a target</span></div>
<div class="body pickwrap">
 <div class="prompt"><span class="usr">bag@plexprep</span>:<span class="pwd">~</span>$ plexprep --serve<span class="cur"></span></div>
 <div class="pkrow"><input id="path" class="pkin" placeholder="paste a folder or file path, or browse below" spellcheck="false">
   <button id="scan" class="selbtn cta">scan &rarr;</button></div>
 <div class="pkrow opt2"><label><input type="checkbox" id="rec" checked> scan subfolders (recursive)</label></div>
 <div class="crumbs" id="crumbs"></div>
 <div class="lsbox" id="ls"></div>
 <div class="done" id="msg">// pick a folder to scan, or type a path and hit scan&nbsp;<span class="cur"></span></div>
</div></div></main>
<script>
(function(){
 function $(id){return document.getElementById(id);}
 function hb(b){b=+b||0; if(b<1024)return b+' B'; var u=['K','M','G','T','P','E'],i=-1,n=b; while(n>=1024&&i<u.length-1){n/=1024;i++;} return n.toFixed(1)+' '+u[i]+'B';}
 function go(){ var p=$('path').value.trim(); if(!p) return; location.href='/report?recursive='+($('rec').checked?'1':'0')+'&path='+encodeURIComponent(p); }
 $('scan').addEventListener('click', go);
 $('path').addEventListener('keydown', function(e){ if(e.key==='Enter') go(); });
 function browse(p){
   fetch('/api/ls?path='+encodeURIComponent(p||'')).then(function(r){return r.json();}).then(function(d){
     if(d.error){ $('msg').textContent='// '+d.error; return; }
     $('path').value=d.path;
     var cb=$('crumbs'); cb.innerHTML='';
     (d.crumbs||[]).forEach(function(c,i){ var a=document.createElement('a'); a.href='#'; a.textContent=c.name; a.onclick=function(e){e.preventDefault();browse(c.path);}; cb.appendChild(a); if(i<d.crumbs.length-1) cb.appendChild(document.createTextNode(' / ')); });
     var ls=$('ls'); ls.innerHTML='';
     if(d.parent!==null){ var up=row('dir','..',''); up.onclick=function(){browse(d.parent);}; ls.appendChild(up); }
     (d.dirs||[]).forEach(function(n){ var r=row('dir',n.name,''); r.onclick=function(){browse(n.path);}; ls.appendChild(r); });
     (d.files||[]).forEach(function(f){ var r=row('file',f.name,hb(f.size)); r.onclick=function(){ location.href='/report?recursive=0&path='+encodeURIComponent(f.path); }; ls.appendChild(r); });
   }).catch(function(e){ $('msg').textContent='// '+e.message; });
 }
 function row(cls,name,meta){ var d=document.createElement('div'); d.className='lsitem '+cls; var a=document.createElement('span'); a.className='nm'; a.textContent=name; var m=document.createElement('span'); m.className='mt'; m.textContent=meta; d.appendChild(a); d.appendChild(m); return d; }
 browse('');
})();
</script></body></html>`
