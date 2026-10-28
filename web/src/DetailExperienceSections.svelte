<script lang="ts">
 import DetailActions from './DetailActions.svelte';import {timeLabel} from './experience';import type {DetailAction,ExperienceView} from './detail-experience';
 export let view:ExperienceView;export let run:(a:DetailAction)=>void;export let allowed:(a:DetailAction)=>boolean;export let busy=false;
</script>
{#each view.sections as section (section.key)}
 <section class="track-section"><div class="shelf-heading"><div><h2>{section.title}</h2>{#if section.help}<p>{section.help}</p>{/if}</div><div class="content-actions"><DetailActions actions={section.actions} {run} {allowed} {busy}/></div></div>
 <ol class="album-tracks">{#each section.rows as row (row.key)}
 {@const main=row.actions.find(a=>a.operation==='open'||a.operation==='play'||a.operation==='unavailable')}<li class:missing-track={!row.fileId}><span class="track-number">{row.ordinal}</span>{#if main}<button class="track-title" disabled={busy||!allowed(main)} onclick={()=>run(main)}><strong>{row.title}</strong>{#if row.subtitle}<small>{row.subtitle}</small>{/if}</button>{:else}<div class="track-title"><strong>{row.title}</strong><small>{row.subtitle}</small></div>{/if}{#if row.duration}<span class="track-duration">{timeLabel(row.duration)}</span>{/if}<div class="episode-actions"><DetailActions actions={row.actions.filter(a=>a.operation!=='open')} {run} {allowed} {busy} compact/></div></li>
 {/each}</ol></section>
{/each}
