<script lang="ts">
 import {onMount} from 'svelte';import ExtensionCard from './ExtensionCard.svelte';import {accountRequest} from './accounts';
 export let token:string;export let refreshKey=0;export let detail=false;export let tab:'installed'|'available'='installed';export let filter='';export let select:(id:string)=>void;
 type State={installed:boolean;enabled:boolean;safeMode:boolean;revision:number;version:string};let state:State|null=null;let ready=false;let busy=false;let error='';let notice='';let confirming=false;
 const name='Informations des fichiers musicaux';const description='Titres, artistes, albums et pochettes intégrés à vos fichiers, lus sans service en ligne.';
 async function load(){state=await accountRequest(token,'audio-tags')}
 async function reload(){try{await load()}catch(e){error=e instanceof Error?e.message:'Extension indisponible.'}}
 onMount(()=>{ready=true});$:if(ready&&refreshKey>=0)void reload();
 async function change(operation:string){if(!state||busy)return;busy=true;error='';notice='';try{await accountRequest(token,'audio-tags','POST',{operation,revision:state.revision});await load();confirming=false;notice=operation==='install'?'Extension installée. Vous pouvez maintenant l’activer.':operation==='enable'?'Extension activée pour les nouveaux fichiers et les fichiers modifiés.':operation==='remove'?'Extension désinstallée. Vos informations et pochettes enregistrées sont conservées.':'Extension désactivée. Vos informations et pochettes enregistrées sont conservées.'}catch(e){error=e instanceof Error?e.message:'Modification indisponible.';await reload()}finally{busy=false}}
</script>
{#if error}<p role="alert" class="extension-warning">{error}</p>{/if}
{#if detail&&state}
 <h3>{name}</h3><p>{description}</p><span class="extension-tag">Mosaune · {state.version} · {state.safeMode?'Suspendu':state.installed?(state.enabled?'Actif':'Désactivé'):'Non installé'}</span>
 {#if notice}<p role="status">{notice}</p>{/if}
 <p>Lors d’un scan musical, cette extension lit les informations déjà présentes dans le morceau. Elle complète la reconnaissance du nom du fichier et fonctionne indépendamment de MusicBrainz.</p>
 <p>Si elle est désactivée ou retirée, les nouveaux morceaux utilisent leur nom de fichier. Les fiches et pochettes déjà enregistrées restent dans votre collection. Réactiver l’extension ne réanalyse pas automatiquement les fichiers inchangés.</p>
 <div class="extension-native-choice"><div><strong>Accès à un seul fichier à la fois</strong><p>Mosaune fournit des blocs de données limités. L’extension ne peut choisir aucun chemin, parcourir vos dossiers ni accéder au réseau.</p></div></div>
 <div class="extension-actions">{#if state.installed}<button class="secondary" disabled={busy||state.safeMode&&!state.enabled} onclick={()=>change(state?.enabled?'disable':'enable')}>{state.enabled?'Désactiver':'Activer'}</button><button class="text-action" disabled={busy} onclick={()=>confirming=true}>Désinstaller</button>{:else}<button class="primary" disabled={busy||state.safeMode} onclick={()=>change('install')}>Installer</button>{/if}</div>
 {#if confirming}<div class="extension-warning"><strong>Désinstaller cette extension ?</strong><p>Son programme sera retiré. Vos médias, métadonnées et pochettes conservés dans Mosaune ne seront pas supprimés.</p><button class="secondary" disabled={busy} onclick={()=>change('remove')}>Confirmer la désinstallation</button><button class="text-action" disabled={busy} onclick={()=>confirming=false}>Annuler</button></div>{/if}
{:else if state&&(tab==='installed'?state.installed:!state.installed)&&(name+' '+description).toLocaleLowerCase().includes(filter.toLocaleLowerCase())}
 <ExtensionCard id="audio-tags" {name} family="Métadonnées · fichiers locaux" {description} status={state.safeMode?'Suspendu':state.installed?(state.enabled?'Actif':'Désactivé'):'À installer'} enabled={state.enabled} disabled={busy||state.safeMode&&!state.enabled} toggle={state.installed?()=>change(state?.enabled?'disable':'enable'):undefined} open={()=>select('audio-tags')}/>
 {#if notice}<p role="status">{notice}</p>{/if}
{/if}
