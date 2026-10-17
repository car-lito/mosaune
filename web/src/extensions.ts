export type Palette = Record<'canvas'|'surface'|'ink'|'muted'|'line'|'accent'|'accentInk'|'sidebar'|'sidebarInk',string>;
export type ExtensionManifest = {schema:number;id:string;name:string;publisher:string;version:string;description:string;family:string;minCore:string;api:number;requires:string[];permissions:string[];theme:{light?:Palette;dark?:Palette;palette?:Record<string,{light:string;dark:string}>;fonts?:Record<string,string[]>;radii?:Record<string,string>;tokens?:Record<string,string>;sections?:Record<string,Record<string,string>>}};
export type ExtensionPackage = {manifest:ExtensionManifest;checksum:string;enabled:boolean;users:string[];preview:{light:Palette;dark:Palette}};
export type ExtensionAppearance = {selected:string;effective:string;stamp:string;safeMode:boolean};
export function appearanceLink(a:ExtensionAppearance){return a.effective&&/^[a-f0-9]{64}$/.test(a.stamp)?'/api/v1/extensions/theme.css?stamp='+a.stamp:'';}
export function extensionsChanged(){window.dispatchEvent(new Event('mosaune:extensions-changed'));}
