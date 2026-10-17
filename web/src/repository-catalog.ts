export type Entry={id:string;name:string;publisher:string;description:string;version:string;kind:string;minCore:string;path:string;sha256:string};
export type Repository={id:string;url:string;ref:string;updatedAt:number;bundled:boolean;official:boolean;index:{name:string;packages:Entry[]}};
export type Choice={repo:Repository;entry:Entry};

export function compareVersions(a:string,b:string){
 const aa=a.split('.').map(Number),bb=b.split('.').map(Number);
 for(let i=0;i<3;i++){if(aa[i]!==bb[i])return aa[i]-bb[i]}
 return 0;
}

export function catalogCandidates(list:Repository[],query:string,repoID:string,kind:string):Choice[]{
 const choices:Choice[]=[];
 for(const repo of list){
  if(repoID&&repo.id!==repoID)continue;
  const latest=new Map<string,Entry>();
  for(const entry of repo.index.packages){
   const key=entry.kind+':'+entry.id,old=latest.get(key);
   if(!old||compareVersions(entry.version,old.version)>0)latest.set(key,entry);
  }
  for(const entry of latest.values()){
   if(kind&&entry.kind!==kind)continue;
   if(!(entry.name+' '+entry.description+' '+entry.publisher).toLocaleLowerCase().includes(query.toLocaleLowerCase()))continue;
   choices.push({repo,entry});
  }
 }
 // A source explicitly selected by the user must remain the download source.
 if(repoID)return choices;
 const grouped=new Map<string,Choice>();
 const priority=(repo:Repository)=>repo.official?2:repo.bundled?1:0;
 for(const candidate of choices){
  const e=candidate.entry;
  // A shared name/ID is not enough: preserve forks and changed or misleading
  // manifests as separate cards. Paths may differ for byte-identical copies.
  const key=JSON.stringify([e.kind,e.id,e.version,e.sha256,e.minCore,e.name,e.publisher,e.description]);
  const old=grouped.get(key);
  if(!old||priority(candidate.repo)>priority(old.repo))grouped.set(key,candidate);
 }
 return [...grouped.values()];
}
