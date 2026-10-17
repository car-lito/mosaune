<script lang="ts">
 export let id:string; export let name:string; export let family:string; export let description:string;
 export let status:string; export let enabled=false; export let toggle:(()=>void)|undefined=undefined;
 export let disabled=false; export let open:()=>void;
</script>
<article class="extension-card extension-summary">
 <button id={'extension-card-'+id} type="button" class="extension-card-open" onclick={open} aria-label={'Ouvrir les paramètres de '+name}>
  <small>{family}</small><h3>{name}</h3><p>{description}</p><span class="extension-tag">{status}</span>
 </button>
 {#if toggle}<button type="button" class="extension-switch" role="switch" aria-checked={enabled} aria-label={'Activer '+name} disabled={disabled} onclick={toggle}><span></span></button>{/if}
</article>
