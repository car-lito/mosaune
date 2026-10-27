<script lang="ts">
 import Portrait from './Portrait.svelte';import {sectionReferences,type SectionBlock,type SectionView} from './detail-sections';import type {Contributor} from './types';
 export let block:SectionBlock;export let view:SectionView;export let person:(provider:string,p:Contributor)=>void;
 let failed:Record<string,boolean>={};
</script>
<section class:cast-section={block.kind==='people'}>
 {#if block.title}{#if block.region==='aside'}<h3>{block.title}</h3>{:else}<h2>{block.title}</h2>{/if}{/if}
 {#if block.text}<p>{block.text}</p>{/if}
 {#if block.kind==='facts'}<dl>{#each block.fields||[] as field}<dt>{field.label}</dt><dd>{field.value}</dd>{/each}</dl>
 {:else if block.kind==='people'}<div class="cast-rail">{#each sectionReferences(block,view.people) as ref}{@const p=ref.resource}<button onclick={()=>person(view.provider,p)}><Portrait name={p.name} image={p.image}/><strong>{p.name}</strong><small>{ref.label||p.role}</small></button>{/each}</div>
 {:else if block.kind==='trailers'}<div class="trailer-links">{#each sectionReferences(block,view.trailers) as ref}{@const trailer=ref.resource}<a class="trailer-card" href={trailer.url} target="_blank" rel="noopener noreferrer"><span class="trailer-thumbnail">{#if trailer.thumbnail&&!failed[trailer.url]}<img src={trailer.thumbnail} alt="" loading="lazy" onerror={()=>failed={...failed,[trailer.url]:true}}/>{/if}<span class="trailer-play">▶</span></span><strong>{trailer.name} ↗</strong><small>{ref.label}</small></a>{/each}</div>{/if}
</section>
