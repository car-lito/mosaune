<script lang="ts">
 export let name:string;
 export let image:string|undefined=undefined;
 export let large=false;
 let failed='';
</script>
<span class="person-avatar" class:large>
 {#if image&&failed!==image}<img src={image} alt={`Portrait de ${name}`} loading="lazy" onerror={()=>failed=image||''}/>{:else}{name.split(' ').map(s=>s[0]).slice(0,2).join('')}{/if}
</span>
