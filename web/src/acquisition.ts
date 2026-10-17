export type DownloadConfig={id:string;url:string;key?:string;configured:boolean;enabled:boolean;profile:number;root:string;library:string};
export type DownloadOptions={profiles:{id:number;name:string}[];roots:{id:number;path:string}[]};
export type DownloadRequest={id:string;workId:string;title:string;kind:string;connector:string;season:number;state:string;message:string;created:number;updated:number;imported:number;total:number;local:number;progress:number;uncertain:boolean};
export const requestStates:Record<string,string>={waiting:'En attente',downloading:'En téléchargement',blocked:'À vérifier',imported:'Importé · dossier à vérifier',available:'Disponible',partial:'Partiellement disponible',unreachable:'Service inaccessible',uncertain:'Envoi à vérifier',missing:'Absent du service',paused:'Suivi en pause',unmonitored:'Surveillance désactivée'};
export async function acquisition<T>(token:string,path:string,method='GET',body?:unknown):Promise<T>{
 const controller=new AbortController();const timeout=setTimeout(()=>controller.abort(),120000);
 try {const r=await fetch('/api/v1/acquisition/'+path,{method,headers:{'Content-Type':'application/json','X-Mosaune-Token':token},body:body===undefined?undefined:JSON.stringify(body),signal:controller.signal});if(r.status===204)return undefined as T;const data=await r.json();if(!r.ok)throw new Error(data.error||'Opération indisponible.');return data;}
 catch(e){if(e instanceof DOMException&&e.name==='AbortError')throw new Error('Réponse trop lente. Consultez Demandes avant de réessayer : un envoi peut avoir été reçu.');throw e;}finally{clearTimeout(timeout);}
}
