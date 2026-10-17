<script lang="ts">
 import {onMount} from 'svelte';
 export let token:string;export let refresh:()=>Promise<void>;
 type Counts={libraryId:string;name:string;total:number;matched:number;review:number;pending:number;failed:number};
 type Status={state:string;paused:boolean;current:string;message:string;retryAt:number;libraries:Counts[];errors:{id:string;title:string;message:string;attempts:number}[]};
 let status:Status|null=null;let failure='';let busy=false;let polling=false;let alive=true;let lastSignature='';let lastRefresh=0;
 const labels:Record<string,string>={idle:'Prêt',complete:'Terminé',paused:'En pause',running:'En cours',waiting:'En cours',unconfigured:'À configurer',retry:'Nouvel essai prévu',errors:'Erreurs à relancer'};
 async function load(){if(polling||document.hidden)return;polling=true;try{
  const r=await fetch('/api/v1/metadata/queue');if(!r.ok)throw new Error('Suivi des fiches indisponible.');const data:Status=await r.json();if(!alive)return;status=data;failure='';
  const signature=JSON.stringify(data.libraries);if(signature!==lastSignature&&(Date.now()-lastRefresh>15000||data.state==='complete')){lastSignature=signature;lastRefresh=Date.now();await refresh();}
 }catch(e){if(alive)failure=e instanceof Error?e.message:'Serveur indisponible.';}finally{polling=false;}}
 async function action(action:string){if(busy)return;busy=true;failure='';try{const r=await fetch('/api/v1/metadata/queue/'+action,{method:'POST',headers:{'X-Mosaune-Token':token}});const data=await r.json();if(!r.ok)throw new Error(data.error);status=data;}catch(e){failure=e instanceof Error?e.message:'Action impossible.';}finally{busy=false;}}
 onMount(()=>{void load();const timer=setInterval(()=>void load(),4000);return()=>{alive=false;clearInterval(timer);};});
 $: failed=status?.libraries.reduce((n,l)=>n+l.failed,0)??0;
</script>
<section class="metadata-queue" aria-labelledby="metadata-progress-title">
 <div class="section-heading"><div><h2 id="metadata-progress-title">Affiches et fiches</h2><p>Un média à la fois, automatiquement. Vous pouvez naviguer pendant l’identification.</p></div>{#if status}<span class="queue-state">{labels[status.state]??status.state}</span>{/if}</div>
 {#if status}
  {#each status.libraries as l}
   <div class="queue-library"><div class="queue-heading"><strong>{l.name}</strong><span>{l.matched+l.review} / {l.total} traités · {Math.floor((l.matched+l.review)*100/Math.max(l.total,1))} %</span></div><progress aria-label={`Fiches traitées : ${l.name}`} max={Math.max(l.total,1)} value={l.matched+l.review}></progress><p>{l.matched} identifiées · {l.review} à vérifier · {l.pending} en attente{l.failed?` · ${l.failed} en erreur`:''}</p></div>
  {:else}<p>Ajoutez un dossier de films ou de séries pour commencer.</p>{/each}
  {#if status.current}<p>Fiche en cours : <strong>{status.current}</strong></p>{/if}
  <p>{status.message}{status.state==='retry'&&status.retryAt?` Nouvel essai au plus tôt à ${new Date(status.retryAt*1000).toLocaleTimeString()}.`:''}</p>
  <div class="queue-actions"><button class="secondary" disabled={busy} onclick={()=>action(status?.paused?'resume':'pause')}>{status.paused?'Reprendre l’identification':'Mettre en pause'}</button>{#if failed}<button class="secondary" disabled={busy} onclick={()=>action('retry')}>Relancer les erreurs</button>{/if}</div>
  {#if status.errors.length}<details><summary>Voir les erreurs ({failed})</summary><p>Jusqu’à trois essais automatiques par fichier. Les autres fiches continuent. Seules les 50 premières erreurs sont détaillées ici.</p>{#each status.errors as e}<p><strong>{e.title}</strong> · {e.attempts} essai{e.attempts>1?'s':''}<br/>{e.message}</p>{/each}</details>{/if}
  <p class="queue-help">Les fiches à vérifier ne sont pas associées au hasard : ouvrez le média puis « Corriger l’association ». La pause et les fiches traitées sont conservées après un redémarrage.</p>
 {/if}
 {#if failure}<p class="playback-error" role="alert">{failure}</p>{/if}
</section>
<style>
 .metadata-queue{padding:2rem 0;border-bottom:1px solid var(--m-metadata-queue-border-bottom)}.queue-state{padding:.6rem 1rem;background:var(--m-queue-state-background);border-radius:var(--m-queue-state-border-radius);white-space:nowrap}.queue-library{padding:1rem 0}.queue-heading,.queue-actions{display:flex;gap:1rem;justify-content:space-between;flex-wrap:wrap}.queue-actions{justify-content:flex-start;margin:1.2rem 0}progress{display:block;width:100%;height:9px;margin:1rem 0;accent-color:var(--m-progress-accent-color)}p{color:var(--m-p-color);line-height:var(--m-p-line-height)}.queue-library p,.queue-help{font-size:var(--m-queue-library-p-font-size)}details{margin:1.5rem 0}summary{cursor:pointer}
</style>
