import {groupMedia} from './collection.ts';
import {groupCard,type Card} from './experience.ts';
import {validPresentation,type ResultPresentation} from './search-presentation.ts';
import type {Entry,Kind,Work,CatalogueSearchEntry} from './types.ts';

// Generic matching only. Media extensions supply candidates, searchable facts,
// priorities and presentation; they cannot supply URLs, IDs or navigation code.
export const normalizeSearch=(text:string)=>text.normalize('NFKD').replace(/\p{M}/gu,'').toLocaleLowerCase().replace(/[^\p{L}\p{N}]+/gu,' ').trim();
const generic:ResultPresentation={version:1,label:'Fichier',preview:{aspectRatio:1,shape:'rectangle',fit:'cover'}};
export function localSearch(entries:Entry[],scope:Kind[],query:string,works:Work[]=[],limit=12):Card[]{
 const term=normalizeSearch(query);if(!term)return [];
 const words=term.split(' '),groups=groupMedia(entries.filter(e=>scope.includes(e.kind)));
 const best=new Map<string,{card:Card;score:number}>();
 for(const group of groups)for(const entry of group.entries){
  const index=entry.catalogue?.search;
  // Old/missing extensions retain only literal filename lookup, never native
  // album/series interpretation. An explicitly empty index means no candidates.
  const specs:CatalogueSearchEntry[]=index?.version===1?index.entries:[{target:'item',title:entry.name,subtitle:'',terms:[entry.name],priority:0,presentation:generic}];
  for(const spec of specs){
   if(!['group','item'].includes(spec.target)||!validPresentation(spec.presentation))continue;
   const title=normalizeSearch(spec.title),haystack=normalizeSearch([spec.title,spec.subtitle,...spec.terms].join(' '));
   if(!words.every(word=>haystack.includes(word)))continue;
   const score=(title===term?1000:title.startsWith(term)?700:title.includes(term)?500:100)+Math.max(0,Math.min(100,spec.priority));
   const card:Card=spec.target==='group'?groupCard(group):{key:'item:'+entry.id,title:entry.name,subtitle:'',kind:entry.kind,local:true,entry,image:entry.posterUrl||(entry.metadata?.imageId?'/api/v1/artwork/'+entry.metadata.imageId:undefined)};
   card.title=spec.title;card.subtitle=spec.subtitle;card.presentation=spec.presentation;
   // The extension explicitly decides whether provider metadata may refine its label.
   const linked=spec.target==='group'&&spec.metadataPresentation?(works.find(w=>group.entries.some(e=>e.workId===w.id))?.metadata.presentation??entry.metadata?.presentation):undefined;
   if(linked&&validPresentation(linked))card.presentation=linked;
   const old=best.get(card.key);if(!old||score>old.score)best.set(card.key,{card,score});
  }
 }
 return [...best.values()].sort((a,b)=>b.score-a.score||a.card.title.localeCompare(b.card.title)||a.card.key.localeCompare(b.card.key)).slice(0,Math.max(0,Math.min(24,limit))).map(v=>v.card);
}
export function mergeSearch(local:Card[],remote:Work[],limit=12):Card[]{
 // Identity comparison is generic; a provider result cannot replace a local card.
 const seen=new Set<string>();
 return remote.filter(w=>{
  const key=JSON.stringify([w.kind,w.metadata.provider,w.metadata.providerId||w.id]);
  if(seen.has(key))return false;seen.add(key);
  return !local.some(c=>(c.group?.entries??(c.entry?[c.entry]:[])).some(e=>e.workId===w.id||(e.kind===w.kind&&e.metadata?.provider===w.metadata.provider&&!!e.metadata?.providerId&&e.metadata.providerId===w.metadata.providerId)));
 }).slice(0,limit).map(w=>({key:w.id,title:w.metadata.title,subtitle:[w.metadata.artist,w.metadata.year].filter(Boolean).join(' · '),kind:w.kind,image:w.posterUrl||(w.poster?'/api/v1/discovery/works/'+w.id+'/poster':undefined),local:w.localIds.length>0,work:w}));
}
