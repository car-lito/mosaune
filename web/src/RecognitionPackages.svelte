<script lang="ts">
 import {onMount} from 'svelte';
 import {accountRequest} from './accounts';
 import ExtensionCard from './ExtensionCard.svelte';
 export let token:string;
 export let tab:'installed'|'available'='installed';
 export let detail='';
 export let filter='';
 export let refreshKey=0;
 export let select:(kind:string)=>void;
 type Package={kind:string;id:string;version:string;installed:boolean;enabled:boolean};
 let packages:Package[]=[];let revision=0;let safeMode=false;let busy=false;let error='';let notice='';let confirming=false;
 let lastRefresh=refreshKey;
 $: if(refreshKey!==lastRefresh){lastRefresh=refreshKey;void load().catch(e=>error=e.message);}
 const names:Record<string,string>={films:'Reconnaissance des films',series:'Reconnaissance des séries',music:'Reconnaissance musicale'};
 $: current=packages.find(p=>p.kind===detail);
 $: visible=packages.filter(p=>p.installed===(tab==='installed')&&names[p.kind].toLocaleLowerCase().includes(filter.toLocaleLowerCase()));
 async function load(){const result=await accountRequest(token,'recognition-packages');packages=result.packages;revision=result.revision;safeMode=result.safeMode;}
 onMount(()=>{void load().catch(e=>error=e.message);const focus=()=>{if(!busy)void load().catch(e=>error=e.message)};window.addEventListener('focus',focus);return()=>window.removeEventListener('focus',focus)});
 async function change(p:Package,operation:string){if(busy)return;busy=true;error='';notice='';try{await accountRequest(token,'recognition-packages/'+p.kind,'POST',{operation,revision});confirming=false;await load();notice=operation==='install'?'Module installé. Activez-le pour reprendre les scans.':operation==='enable'?'Reconnaissance activée. Vous pouvez actualiser vos bibliothèques.':operation==='disable'?'Reconnaissance désactivée. Les analyses non enregistrées sont interrompues ; votre catalogue est conservé.':'Module désinstallé. Votre catalogue est conservé. Vous pourrez le réinstaller depuis les extensions disponibles.';}catch(e){error=e instanceof Error?e.message:'Modification indisponible.';try{await load()}catch{}}finally{busy=false}}
</script>
{#if error}<p class="form-error" role="alert">{error}</p>{/if}
{#if notice}<p class="extension-notice" role="status">{notice}</p>{/if}
{#if detail}
 {#if current}
  <article class="extension-detail"><div class="extension-card-body">
   <span class="extension-tag">Reconnaissance · {current.version} · {current.installed?(safeMode?'Suspendu':current.enabled?'Actif':'Désactivé'):'Non installé'}</span>
   <p>Reconnaît les formats, les titres et les numéros d’épisodes à partir des noms de fichiers. Ce module s’exécute uniquement pendant les analyses, sans accès au réseau.</p>
   <p>La désactivation arrête les analyses non enregistrées de ce type de média. Une sauvegarde déjà commencée se termine avant le changement d’état. Les médias, affiches et progressions déjà enregistrés restent disponibles.</p>
   <p>Il s’agit du module de reconnaissance : les écrans, les lecteurs et les autres fonctions médias sont encore intégrés à Mosaune.</p>
   <div class="extension-actions">
    {#if current.installed}
     <button class="secondary" disabled={busy||safeMode&&!current.enabled} onclick={()=>current&&change(current,current.enabled?'disable':'enable')}>{current.enabled?'Désactiver':'Activer'}</button>
     <button class="text-action" disabled={busy} onclick={()=>confirming=true}>Désinstaller</button>
    {:else}<button class="secondary" disabled={busy||safeMode} onclick={()=>current&&change(current,'install')}>Installer la version fournie avec Mosaune</button>{/if}
   </div>
   {#if confirming&&current.installed}<div class="extension-warning"><strong>Désinstaller ce module ?</strong><p>Son programme installé sera retiré. Les nouveaux scans seront indisponibles jusqu’à sa réinstallation. Vos fichiers et votre collection sont conservés.</p><div class="extension-actions"><button class="secondary" disabled={busy} onclick={()=>current&&change(current,'remove')}>Confirmer la désinstallation</button><button class="text-action" disabled={busy} onclick={()=>confirming=false}>Annuler</button></div></div>{/if}
  </div></article>
 {:else}<p role="status">Chargement du module…</p>{/if}
{:else if tab==='installed'}
 {#each visible as p (p.id)}<ExtensionCard id={'recognition.'+p.kind} name={names[p.kind]} family="Médias · reconnaissance" description="Identification locale des noms et formats pendant les scans." status={safeMode?'Suspendu':p.enabled?'Actif':'Désactivé'} enabled={p.enabled&&!safeMode} disabled={busy||safeMode&&!p.enabled} toggle={()=>change(p,p.enabled?'disable':'enable')} open={()=>select(p.kind)}/>{/each}
{:else if visible.length}
 <section><h3>Modules fournis avec Mosaune</h3><p>Ces modules sont disponibles ici sans téléchargement. L’installation ne les active pas automatiquement.</p><div class="extension-grid">{#each visible as p (p.id)}<ExtensionCard id={'recognition.'+p.kind} name={names[p.kind]} family="Médias · reconnaissance" description="Reconnaissance locale des noms et formats. Aucun accès au réseau." status="À installer" open={()=>select(p.kind)}/>{/each}</div></section>
{/if}
