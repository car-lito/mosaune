<script lang="ts">
 import {acquisition,type DownloadConfig,type DownloadOptions} from './acquisition';
 export let initial:DownloadConfig;export let token:string;export let libraries:{id:string;name:string;kind:string;path?:string}[];export let refresh:()=>Promise<void>;
 let config:DownloadConfig={...initial,key:''};let options:DownloadOptions|undefined;let busy=false;let message='';let error='';let verified='';
 let addingFolder=false;let folderPath='';let folderName='';let addedFolders:typeof libraries=[];
 $: name=config.form?.title??config.id;
 $: kind=config.id==='radarr'?'films':'series';
 $: folderLabel=kind==='films'?'films':'séries';
 $: folders=[...libraries,...addedFolders.filter(a=>!libraries.some(l=>l.id===a.id))].filter(l=>l.kind===kind);
 $: selectedFolder=folders.find(l=>l.id===config.library);
 $: signature=config.url+'\n'+config.key;
 function payload(){const {form,formError,...input}=config;return input;}
 async function reload(){busy=true;error='';try{const all=await acquisition<DownloadConfig[]>(token,'settings');const current=all.find(c=>c.id===config.id);if(!current)throw new Error('Connecteur indisponible.');config={...current,key:''};options=undefined;verified='';}catch(e){error=e instanceof Error?e.message:'Réglages indisponibles.';}finally{busy=false;}}
 async function test(){busy=true;message='';error='';try{const data=await acquisition<{config:DownloadConfig;options:DownloadOptions}>(token,'test','POST',payload());options=data.options;config={...data.config,key:config.key};verified=config.url+'\n'+config.key;if(!options.profiles.some(p=>p.id===config.profile))config.profile=options.profiles[0]?.id??0;if(!options.roots.some(p=>p.path===config.root))config.root=options.roots[0]?.path??'';message='Connexion réussie. Aucun téléchargement demandé.';}catch(e){error=e instanceof Error?e.message:'Connexion impossible.';options=undefined;}finally{busy=false;}}
 async function save(){busy=true;message='';error='';try{const data=await acquisition<{config:DownloadConfig;options:DownloadOptions}>(token,'settings','PUT',{...payload(),enabled:true});config={...data.config,key:''};options=data.options;verified=config.url+'\n';message='Réglages enregistrés. Le connecteur est actif.';}catch(e){error=e instanceof Error?e.message:'Enregistrement impossible.';}finally{busy=false;}}
 async function pause(){busy=true;error='';try{await acquisition(token,config.id+'/pause','POST',{});config.enabled=false;message='Suivi Mosaune en pause. Les téléchargements déjà envoyés continuent dans '+name+'.';}catch(e){error=e instanceof Error?e.message:'Pause impossible.';}finally{busy=false;}}
 async function addFolder(){
  if(busy)return;busy=true;message='';error='';
  try{
   const response=await fetch('/api/v1/libraries',{method:'POST',headers:{'Content-Type':'application/json','X-Mosaune-Token':token},body:JSON.stringify({name:folderName.trim(),path:folderPath.trim(),kind})});
   const result=await response.json();if(!response.ok)throw new Error(result.error||'Impossible d’ajouter ce dossier.');
   addedFolders=[...addedFolders,{id:result.id,name:result.name,kind,path:result.path}];config.library=result.id;addingFolder=false;folderPath='';folderName='';
   message=`Dossier « ${result.name} » ajouté et sélectionné. Cliquez sur « Enregistrer et activer » pour terminer la configuration.`;
   try{await refresh();}catch{error='Le dossier a été ajouté, mais la bibliothèque affichée n’a pas pu être actualisée. Vous pouvez terminer la configuration puis recharger la page.';}
  }catch(e){error=e instanceof Error?e.message:'Le serveur ne répond pas. Vérifiez Vos dossiers avant de réessayer.';}finally{busy=false;}
 }
</script>
<section class="download-connector">
 <div class="download-heading"><h3>{name} <small>· {config.id==='radarr'?'Films':'Séries'}</small></h3><span class="download-badge">{config.enabled?'Actif':'À configurer / en pause'}</span></div>
 {#if config.form}
 {#if config.form.instructions.length}<details class="download-guide"><summary>Configurer {name}</summary><ol>{#each config.form.instructions as instruction}<li>{instruction}</li>{/each}</ol></details>{/if}
 <div class="download-fields">
  {#each config.form.fields.filter(f=>f.binding==='url'||f.binding==='key') as field (field.binding)}
   <label>{field.label}<input type={field.binding==='key'?'password':'url'} value={field.binding==='key'?config.key:config.url} oninput={e=>{if(field.binding==='key')config.key=e.currentTarget.value;else config.url=e.currentTarget.value;}} disabled={busy} maxlength={field.maxLength} placeholder={field.binding==='key'&&config.configured?'Déjà enregistrée · laisser vide pour la garder':field.placeholder} autocomplete={field.binding==='key'?'new-password':'off'}/>{#if field.help}<small>{field.help}</small>{/if}</label>
  {/each}
 </div>
 <button class="secondary" disabled={busy||!config.url} onclick={test}>{busy?'Patientez…':'Tester la connexion'}</button>
 {#if options&&verified===signature}
  <div class="download-fields">
   {#each config.form.fields.filter(f=>f.binding!=='url'&&f.binding!=='key') as field (field.binding)}
    <div class="download-folder-choice">
     {#if field.binding==='profile'}
      <label>{field.label}<select bind:value={config.profile} disabled={busy}>{#each options.profiles as p}<option value={p.id}>{p.name}</option>{/each}</select></label>
     {:else if field.binding==='root'}
      <label>{field.label}<select bind:value={config.root} disabled={busy}>{#each options.roots as p}<option value={p.path}>{p.path}</option>{/each}</select></label>
     {:else if field.binding==='library'}
      <label>{field.label}<select bind:value={config.library} disabled={busy||!folders.length}><option value="">{folders.length?'Choisir un dossier déjà ajouté…':'Aucun dossier de '+folderLabel+' ajouté'}</option>{#each folders as l}<option value={l.id}>{l.name}{l.path?' — '+l.path:''}</option>{/each}</select></label>
      {#if selectedFolder?.path}<p class="folder-help download-folder-path">{selectedFolder.path}</p>{/if}
      <button class="text-action" disabled={busy} aria-expanded={addingFolder} onclick={()=>addingFolder=!addingFolder}>{addingFolder?'Fermer l’ajout':'+ Ajouter un dossier de '+folderLabel}</button>
     {/if}
     {#if field.help}<p class="folder-help">{field.help}</p>{/if}
    </div>
   {/each}
  </div>
  {#if addingFolder}
   <form class="download-folder-form" onsubmit={e=>{e.preventDefault();void addFolder();}}>
    <h4>Ajouter votre dossier de {folderLabel}</h4>
    <p>Dans l’Explorateur Windows, ouvrez le dossier du NAS qui contient vos {folderLabel}, puis copiez son adresse avec <strong>Ctrl+L, Ctrl+C</strong> et collez-la ci-dessous. Le dossier doit être accessible depuis l’ordinateur qui exécute Mosaune.</p>
    <div class="download-fields"><label>Chemin complet du dossier<input required bind:value={folderPath} disabled={busy} placeholder={kind==='films'?'\\\\NAS\\Medias\\Films':'\\\\NAS\\Medias\\Series'}/><small>Exemple de partage réseau ; utilisez le nom et le chemin réels de votre NAS. Le chemin « {config.root} » appartient à {name} et peut être différent sur ce PC.</small></label><label>Nom dans Mosaune (facultatif)<input bind:value={folderName} maxlength="150" disabled={busy} placeholder={kind==='films'?'Films du NAS':'Séries du NAS'}/></label></div>
    <button class="primary" disabled={busy||!folderPath.trim()}>{busy?'Analyse en cours…':'Ajouter et sélectionner'}</button>
    {#if busy}<p class="folder-help" role="status">Mosaune analyse le dossier sans déplacer vos fichiers. Gardez cette page ouverte jusqu’à la fin.</p>{/if}
   </form>
  {/if}
  <p class="folder-help">Choisissez le dossier Mosaune qui permet de lire les fichiers une fois importés. Sur votre PC, il peut s’agir du partage réseau du NAS. Les nouveaux ajouts utiliseront le profil et la destination ci-dessus ; les œuvres déjà présentes dans {name} garderont les leurs.</p>
  <button class="primary" disabled={busy||!selectedFolder||!config.root||!config.profile} onclick={save}>Enregistrer et activer</button>
 {/if}
 {:else}<p class="playback-error" role="alert">{config.formError||'Formulaire du connecteur indisponible.'}</p>{/if}
 <button class="text-action" disabled={busy} onclick={reload}>Recharger les réglages</button>
 {#if config.enabled}<button class="text-action" disabled={busy} onclick={pause}>Mettre le suivi en pause</button>{/if}
 {#if message}<p class="notice" role="status">{message}</p>{/if}{#if error}<p class="playback-error" role="alert">{error}</p>{/if}
</section>
