<script lang="ts">
 import {onMount} from 'svelte';
 type Scan={name:string;state:string;found:number;total:number;processed:number;message:string;updated:number};
 let scan:Scan|null=null;let unavailable=false;let polling=false;let alive=true;
 async function load(){if(polling||document.hidden)return;polling=true;try{const r=await fetch('/api/v1/library/scan');if(!r.ok)throw new Error();const data=await r.json();if(alive){scan=data;unavailable=false;}}catch{if(alive)unavailable=true;}finally{polling=false;}}
 onMount(()=>{void load();const timer=setInterval(()=>void load(),1000);return()=>{alive=false;clearInterval(timer);};});
 $: percent=scan?.total?Math.floor(scan.processed*100/scan.total):0;
</script>
{#if scan&&scan.state!=='idle'&&scan.state!==''}
 <div class="scan-progress" aria-label="Progression de l’analyse des fichiers">
  <div class="scan-heading"><strong>Analyse des fichiers · {scan.name||'Nouveau dossier'}</strong><span>{scan.state==='complete'?'Terminée':scan.state==='failed'?'Interrompue':scan.state==='discovering'?'Découverte du dossier':`${percent} %`}</span></div>
  {#if scan.state==='discovering'}<progress aria-label="Découverte des fichiers"></progress><p>{scan.found} fichier{scan.found>1?'s':''} trouvé{scan.found>1?'s':''}. Le total est en cours de calcul.</p>
  {:else}<progress aria-label={scan.state==='recognizing'?'Reconnaissance des noms':'Enregistrement des fichiers'} max={Math.max(scan.total,1)} value={scan.state==='complete'&&scan.total===0?1:scan.processed}></progress><p>{scan.state==='recognizing'?'Reconnaissance des noms':'Enregistrement dans la bibliothèque'} · {scan.processed} / {scan.total}{scan.state==='indexing'&&scan.processed===scan.total?' · Finalisation…':''}. {scan.state==='complete'?'Les affiches et fiches sont traitées séparément, ci-dessous.':''}</p>{/if}
  {#if scan.state==='failed'}<p class="playback-error" role="alert">{scan.message}</p>{/if}
  {#if unavailable}<p role="status">Le suivi ne répond plus. L’avancement affiché est le dernier reçu.</p>{/if}
 </div>
{/if}
<style>
 .scan-progress{padding:1.25rem;margin:1rem 0;background:var(--m-scan-progress-background);border:1px solid var(--m-scan-progress-border);border-radius:var(--m-scan-progress-border-radius)}.scan-heading{display:flex;justify-content:space-between;gap:1rem;flex-wrap:wrap}progress{display:block;width:100%;height:9px;margin:1rem 0;accent-color:var(--m-progress-accent-color)}p{font-size:var(--m-p-font-size);margin:.5rem 0;color:var(--m-p-color)}
</style>
