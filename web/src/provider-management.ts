export type ProviderManagement = {
 id:string; name:string; enabled:boolean; available:boolean; activation:boolean;
 packages:boolean; runtimeControl:boolean; configurationOwner?:string; runtime?:Record<string,boolean>;
};
export function providerStatus(p:ProviderManagement):string {
 if(p.runtime?.installed===false)return 'Non installé';
 if(!p.enabled)return 'Désactivé';
 if(p.runtime?.safeMode)return 'Mode de secours';
 if(p.runtime?.enabled===false)return 'Module en pause';
 if(p.runtime?.serviceAccess===false)return 'Accès en ligne retiré';
 return p.available?'Actif':'Indisponible';
}
// Which settings surface a provider gets. Forms come only from the schema the
// provider declares; no provider name selects a built-in form or guide.
export type ProviderSettingsView={kind:'declared';owner:string}|{kind:'missing'}|{kind:'none'};
export function providerSettingsView(p:{configurationOwner?:string;requiresCredential:boolean}):ProviderSettingsView{
 if(p.configurationOwner)return {kind:'declared',owner:p.configurationOwner};
 return p.requiresCredential?{kind:'missing'}:{kind:'none'};
}
export function providerRoute(id:string, action:'state'|'runtime'):string {
 return 'metadata/providers/'+encodeURIComponent(id)+'/'+action;
}
export function providerPackageRoute(id:string, action?:'preview'|'install'|'rollback'):string {
 return 'provider-packages/'+encodeURIComponent(id)+(action?'/'+action:'');
}
