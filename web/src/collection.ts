import type { Entry, Kind } from './types.ts';
export type Group = { key: string; title: string; subtitle: string; kind: Kind; imageId?: string; entries: Entry[] };
export const compareCatalogueEntries=(a:Entry,b:Entry)=>(a.catalogue?.order[0]??0)-(b.catalogue?.order[0]??0)||(a.catalogue?.order[1]??0)-(b.catalogue?.order[1]??0)||(a.catalogue?.sortText??a.name).localeCompare(b.catalogue?.sortText??b.name)||a.id.localeCompare(b.id);
// The renderer composes server projections without knowing album/episode rules.
export function groupMedia(entries: Entry[]): Group[] {
 const groups=new Map<string,Group>();
 for(const entry of entries){const p=entry.catalogue;
 const key=p?.key?JSON.stringify([entry.kind,p.owner,p.key]):JSON.stringify(['item',entry.id]);
 const old=groups.get(key);if(old){old.entries.push(entry);old.imageId ||= entry.metadata?.imageId;}
 else groups.set(key,{key,title:p?.title||entry.name,subtitle:p?.subtitle||'',kind:entry.kind,imageId:entry.metadata?.imageId,entries:[entry]});
 }
 for(const g of groups.values())g.entries.sort(compareCatalogueEntries);
 return [...groups.values()].sort((a,b)=>a.title.localeCompare(b.title,'fr'));
}
