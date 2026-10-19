<script lang="ts">
 import {onMount} from 'svelte';
 import {accountRequest} from './accounts';
 export let id:string;export let token:string;export let refresh:()=>Promise<void>;
 type Field={id:string;label:string;type:'text'|'secret'|'boolean'|'select';help?:string;minLength?:number;maxLength?:number;options?:{value:string;label:string}[]};
 type Configuration={revision:number;schema:{version:number;title:string;description:string;instructions?:string[];links?:{label:string;url:string}[];fields:Field[]};values:Record<string,string|boolean>;secretsConfigured:Record<string,boolean>;services:{name:string;label:string;description:string}[]};
 let config:Configuration|null=null;let values:Record<string,string|boolean>={};let busy=false;let error='';let message='';let clearing='';
 const route='extensions/'+encodeURIComponent(id)+'/configuration';
 async function load(){config=await accountRequest(token,route);values={...config!.values};}
 async function reload(){if(busy)return;busy=true;error='';try{await load();clearing='';message='Réglages actualisés.';}catch(e){error=e instanceof Error?e.message:'Actualisation impossible.';}finally{busy=false;}}
 onMount(()=>{void load().catch(e=>error=e instanceof Error?e.message:'Réglages indisponibles.');});
 async function save(clearSecrets:string[]=[]){
  if(busy||!config)return;busy=true;error='';message='';
  const changes:Record<string,string|boolean>={};
  if(!clearSecrets.length)for(const field of config.schema.fields){const value=values[field.id];if(value!==undefined&&(field.type!=='secret'||value!==''))changes[field.id]=value;}
  try{
   await accountRequest(token,route,'PUT',{values:changes,clearSecrets,revision:config.revision});
   values={};clearing='';await load();window.dispatchEvent(new Event('mosaune-provider-settings'));message=clearSecrets.length?'Clé retirée. Les fiches déjà enregistrées restent disponibles.':'Réglages enregistrés.';
   // Library refresh is independent: an unavailable listing must not turn a
   // successful credential save into a misleading save error.
   void refresh().catch(()=>{});
  }catch(e){error=e instanceof Error?e.message:'Enregistrement impossible.';}finally{busy=false;}
 }
</script>
<section class="extension-configuration" aria-busy={busy}>
 {#if config}
  <h3>{config.schema.title}</h3><p>{config.schema.description}</p>
  {#if config.schema.instructions?.length}<ol>{#each config.schema.instructions as line}<li>{line}</li>{/each}</ol>{/if}
  {#if config.schema.links?.length}<div class="extension-actions">{#each config.schema.links as link}<a class="text-action" href={link.url} target="_blank" rel="noreferrer">{link.label} ↗</a>{/each}</div>{/if}
  <form onsubmit={e=>{e.preventDefault();void save();}}>
   {#each config.schema.fields as field (field.id)}
    <div class="extension-config-field">
     {#if field.type==='boolean'}
      <label class="extension-config-check"><input type="checkbox" checked={values[field.id]===true} onchange={e=>values[field.id]=e.currentTarget.checked} disabled={busy}/>{field.label}</label>
     {:else if field.type==='select'}
      <label>{field.label}<select value={String(values[field.id]??'')} onchange={e=>values[field.id]=e.currentTarget.value} disabled={busy}><option value="" disabled>Choisir…</option>{#each field.options??[] as option}<option value={option.value}>{option.label}</option>{/each}</select></label>
     {:else}
      <label>{field.label}<input type={field.type==='secret'?'password':'text'} autocomplete="off" value={String(values[field.id]??'')} oninput={e=>values[field.id]=e.currentTarget.value} minlength={field.minLength} maxlength={field.maxLength} disabled={busy} placeholder={field.type==='secret'&&config.secretsConfigured[field.id]?'Laisser vide pour conserver la clé':''}/></label>
     {/if}
     {#if field.help}<small>{field.help}</small>{/if}
     {#if field.type==='secret'}<div class="extension-actions"><span class="extension-tag">{config.secretsConfigured[field.id]?'Clé enregistrée':'Non connecté'}</span>{#if config.secretsConfigured[field.id]}<button type="button" class="text-action" disabled={busy} onclick={()=>clearing=field.id}>Retirer la clé</button>{/if}</div>{/if}
     {#if clearing===field.id}<div class="extension-warning"><p>Retirer cette clé du serveur ? Le fournisseur ne pourra plus récupérer de nouvelles informations tant qu’une clé n’aura pas été enregistrée.</p><div class="extension-actions"><button type="button" class="secondary" disabled={busy} onclick={()=>save([field.id])}>Confirmer le retrait</button><button type="button" class="text-action" disabled={busy} onclick={()=>clearing=''}>Annuler</button></div></div>{/if}
    </div>
   {/each}
   <button class="secondary" disabled={busy}>{busy?'Enregistrement…':'Enregistrer les réglages'}</button>
  </form>
  {#if config.services.length}<div class="extension-access"><h3>Accès du module</h3>{#each config.services as service}<strong>{service.label}</strong><p>{service.description}</p>{/each}</div>{/if}
 {:else if !error}<p role="status">Chargement des réglages…</p>{/if}
 {#if error}<p class="form-error" role="alert">{error}</p><button type="button" class="secondary" disabled={busy} onclick={reload}>Recharger les réglages</button>{/if}
 {#if message}<p class="extension-notice" role="status">{message}</p>{/if}
</section>
