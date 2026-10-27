import type {Contributor} from './types';
export type SectionBlock={key:string;region:'main'|'aside'|'full';kind:'paragraph'|'facts'|'people'|'trailers';title:string;text?:string;fields?:{label:string;value:string}[];references?:{index:number;label:string}[]};
export type SectionView={owner:string;provider:string;blocks:SectionBlock[];people:Contributor[];trailers:{name:string;url:string;thumbnail?:string;language?:string}[]};
export function regionBlocks(view:SectionView|null,region:SectionBlock['region']){return view?.blocks.filter(b=>b.region===region)||[];}
// Resolve host-owned references; an absent resource never creates a clickable target.
export function sectionReferences<T>(block:SectionBlock,resources:T[]){return (block.references||[]).flatMap(ref=>resources[ref.index]?[{...ref,resource:resources[ref.index]}]:[]);}
