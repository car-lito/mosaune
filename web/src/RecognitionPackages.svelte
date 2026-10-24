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
 type Package={kind:string;id:string;version:string;installed:boolean;enabled:boolean;name:string;publisher:string;factory:boolean;selected:boolean;previousVersion:string};
 let packages:Package[]=[];let revision=0;let safeMode=false;let busy=false;let error='';let notice='';let confirming=false;
 let lastRefresh=refreshKey;
 $: if(refreshKey!==lastRefresh){lastRefresh=refreshKey;void load().catch(e=>error=e.message);}
 const names:Record<string,string>={films:'Reconnaissance des films',series:'Reconnaissance des séries',music:'Reconnaissance musicale'};
 $: current=packages.find(p=>p.id===detail||p.factory&&p.kind===detail);
 $: visible=packages.filter(p=>p.installed===(tab==='installed')&&(p.name+' '+names[p.kind]).toLocaleLowerCase().includes(filter.toLocaleLowerCase()));
 async function load(){const result=await accountRequest(token,'recognition-packages');packages=result.packages;revision=result.revision;safeMode=result.safeMode;}
 onMount(()=>{void load().catch(e=>error=e.message);const focus=()=>{if(!busy)void load().catch(e=>error=e.message)};window.addEventListener('focus',focus);return()=>window.removeEventListener('focus',focus)});
 async function change(p:Package,operation:string){if(busy)return;busy=true;error='';notice='';try{await accountRequest(token,'recognition-packages/'+encodeURIComponent(p.id),'POST',{operation,revision});confirming=false;await load();const messages:Record<string,string>={install:'Module installé. Activez-le puis choisissez-le pour les scans.',enable:'Module activé. Le choix pour les scans est indiqué sur sa fiche.',disable:'Module désactivé. Votre catalogue est conservé.',select:'Ce module sera utilisé pour les prochains scans. Les analyses non enregistrées de ce média ont été interrompues.',restore:'Version précédente restaurée. Votre choix et votre collection sont conservés.',remove:'Module désinstallé. Votre catalogue est conservé. Choisissez un autre module pour poursuivre les scans.'};notice=messages[operation];if(operation==='remove'&&!p.factory)select('');}catch(e){error=e instanceof Error?e.message:'Modification indisponible.';try{await load()}catch{}}finally{busy=false}}
 function status(p:Package){return safeMode?'Suspendu':!p.enabled?'Désactivé':p.selected?'Utilisé pour les scans':'Actif · non sélectionné'}
</script>
{#if error}<p class="form-error" role="alert">{error}</p>{/if}
{#if notice}<p class="extension-notice" role="status">{notice}</p>{/if}
{#if detail}
 {#if current}
  <article class="extension-detail"><div class="extension-card-body">
   <h3>{current.name}</h3>
   <span class="extension-tag">Reconnaissance · {current.version} · {current.installed?status(current):'Non installé'}</span>
   <p>{current.factory?'Fourni avec Mosaune':'Module externe · éditeur déclaré : '+current.publisher+' (non vérifié)'}</p>
   <p>Un seul reconnaisseur est choisi par type de média. En désactivant ou retirant le module choisi, les scans sont suspendus jusqu’à sa réactivation ou au choix d’un remplaçant.</p>
   <p>Reconnaît les formats, les titres et les numéros d’épisodes à partir des noms de fichiers. Ce module s’exécute uniquement pendant les analyses, sans accès au réseau.</p>
   <p>La désactivation arrête les analyses non enregistrées de ce type de média. Une sauvegarde déjà commencée se termine avant le changement d’état. Les médias, affiches et progressions déjà enregistrés restent disponibles.</p>
   <p>Il s’agit du module de reconnaissance : les écrans, les lecteurs et les autres fonctions médias sont encore intégrés à Mosaune.</p>
   <div class="extension-actions">
    {#if current.installed}
     <button class="secondary" disabled={busy||safeMode&&!current.enabled} onclick={()=>current&&change(current,current.enabled?'disable':'enable')}>{current.enabled?'Désactiver':'Activer'}</button>
     {#if !current.selected}<button class="secondary" disabled={busy||safeMode||!current.enabled} onclick={()=>current&&change(current,'select')}>Utiliser pour les scans</button>{/if}
     {#if current.previousVersion}<button class="secondary" disabled={busy||safeMode} onclick={()=>current&&change(current,'restore')}>Restaurer la version {current.previousVersion}</button>{/if}
     <button class="text-action" disabled={busy} onclick={()=>confirming=true}>Désinstaller</button>
    {:else}<button class="secondary" disabled={busy||safeMode} onclick={()=>current&&change(current,'install')}>Installer la version fournie avec Mosaune</button>{/if}
   </div>
   {#if confirming&&current.installed}<div class="extension-warning"><strong>Désinstaller ce module ?</strong><p>Son programme installé sera retiré. Les nouveaux scans seront indisponibles jusqu’à sa réinstallation. Vos fichiers et votre collection sont conservés.</p><div class="extension-actions"><button class="secondary" disabled={busy} onclick={()=>current&&change(current,'remove')}>Confirmer la désinstallation</button><button class="text-action" disabled={busy} onclick={()=>confirming=false}>Annuler</button></div></div>{/if}
  </div></article>
 {:else}<p role="status">Chargement du module…</p>{/if}
{:else if tab==='installed'}
 {#each visible as p (p.id)}<ExtensionCard id={'recognition.'+p.id} name={p.name} family="Médias · reconnaissance" description={p.factory?'Identification locale des noms et formats pendant les scans.':'Module externe · '+p.publisher+' (éditeur non vérifié)'} status={status(p)} enabled={p.enabled&&!safeMode} disabled={busy||safeMode&&!p.enabled} toggle={()=>change(p,p.enabled?'disable':'enable')} open={()=>select(p.id)}/>{/each}
{:else if visible.length}
 <section><h3>Modules fournis avec Mosaune</h3><p>Ces modules sont disponibles ici sans téléchargement. L’installation ne les active pas automatiquement.</p><div class="extension-grid">{#each visible as p (p.id)}<ExtensionCard id={'recognition.'+p.kind} name={names[p.kind]} family="Médias · reconnaissance" description="Reconnaissance locale des noms et formats. Aucun accès au réseau." status="À installer" open={()=>select(p.kind)}/>{/each}</div></section>
{/if}

