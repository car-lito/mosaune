<script lang="ts">
  import { onMount, tick } from 'svelte';
 import ExtensionSettings from './ExtensionSettings.svelte';
 import {appearanceLink,type ExtensionAppearance} from './extensions';
 let extensionAppearance:ExtensionAppearance={selected:'',effective:'',stamp:'',safeMode:false};
 async function refreshAppearance(){try{const r=await fetch('/api/v1/extensions/appearance');if(r.ok)extensionAppearance=await r.json();else extensionAppearance={selected:'',effective:'',stamp:'',safeMode:false};}catch{extensionAppearance={selected:'',effective:'',stamp:'',safeMode:false};}}
 import AccountSettings from './AccountSettings.svelte';import {permitted,type Account} from './accounts';
 export let user:Account;export let onlogout:()=>void;export let sessionEnded:()=>void;export let accountChanged:(a:Account)=>void;
  import Icon from './Icon.svelte';
  import LibraryManager from './LibraryManager.svelte';import ExtensionPage from './ExtensionPage.svelte';
 import {pageKey as uiPageKey,screenPage,type UIPage} from './ui-pages';
 let navigation:UIPage[]=[];let uiRevision=0;let externalPage:{owner:string;id:string}|null=null;let externalTitle='';let navigationRequest=0;
 async function refreshNavigation(){const request=++navigationRequest;try{const r=await fetch('/api/v1/ui/navigation');if(!r.ok)throw Error('navigation');const data:UIPage[]=await r.json();if(request!==navigationRequest)return;navigation=data;if(settingsTab.startsWith('extension:')&&!data.some(p=>p.placement==='settings'&&uiPageKey(p)===settingsTab))settingsTab='extensions';uiRevision++;}catch{if(request===navigationRequest){navigation=[];uiRevision++;}}}
 function openUI(p:UIPage){if(p.screen){navigate(p.screen as View);return;}if(!p.owner){if(p.placement==='settings'){navigate('settings');settingsTab=p.id;}else navigate(p.id as View);return;}externalTitle=p.label;if(p.placement==='settings'){navigate('settings');settingsTab=uiPageKey(p);}else{navigate('home');externalPage={owner:p.owner,id:p.id};}}
 function uiNavigate(target:string,owner:string){if(target.startsWith('page:')){navigate('home');externalPage={owner,id:target.slice(5)};return;}const item=navigation.find(p=>p.screen===target||!p.owner&&(p.placement==='settings'?'settings:'+p.id:p.id)===target);if(item)openUI(item);}
 $: activeScreen=screenPage(navigation,view);
 $: settingsSections=navigation.filter(p=>p.placement==='settings');
 $: settingsPage=settingsSections.find(p=>uiPageKey(p)===settingsTab&&p.owner);
 let settingsTab='account';
  import MediaPlayer from './MediaPlayer.svelte';
  import MediaCollection from './MediaCollection.svelte';
  import MetadataSettings from './MetadataSettings.svelte';
 import MetadataQueue from './MetadataQueue.svelte';
  import MetadataDetails from './MetadataDetails.svelte';
  import PlaylistManager from './PlaylistManager.svelte';
  import QueuePanel from './QueuePanel.svelte';
  import SearchBox from './SearchBox.svelte';
 import {acquisition,type DownloadRequest} from './acquisition';
 import DownloadSettings from './DownloadSettings.svelte';import DownloadRequests from './DownloadRequests.svelte';
 import ContentPage from './ContentPage.svelte';import PersonPage from './PersonPage.svelte';
 import {groupCard,type Card} from './experience';import {waitForProgress} from './progress';import type {Work,Contributor} from './types';
  import { emptyQueue, makeQueue, appendQueue, shuffleQueue, moveItem, nextInQueue, savedQueue, restoreQueue, type MusicQueue, type QueueItem } from './music';
  import { groupMedia, compareCatalogueEntries } from './collection';
  import type { Kind, Entry, Provider } from './types';

  type View = Kind | 'home' | 'watchlist' | 'favorites' | 'playlists' | 'settings' | 'requests';
 type Page = {type:'content';card:Card}|{type:'person';provider:string;id:string;name:string};
 let page:Page|null=null;let backStack:{view:View;page:Page|null;externalPage:{owner:string;id:string}|null}[]=[];let works:Work[]=[];let pageKey=0;
 async function openCard(card:Card){backStack=[...backStack,{view,page,externalPage}];externalPage=null;view=card.kind;page={type:'content',card};pageKey++;window.scrollTo({top:0});}
 function openPerson(provider:string,p:Contributor){backStack=[...backStack,{view,page,externalPage}];externalPage=null;page={type:'person',provider,id:p.id,name:p.name};pageKey++;window.scrollTo({top:0});}
 function playAlbum(card:Card){const tracks=card.group?.entries??entries.filter(e=>card.work?.localIds.includes(e.id));playMusic([...tracks].sort(compareCatalogueEntries));}
 function openArtist(name:string){openPerson('',{id:'',name});}
 function back(){const previous=backStack.pop();backStack=[...backStack];page=previous?.page??null;view=previous?.view??'home';externalPage=previous?.externalPage??null;pageKey++;window.scrollTo({top:0});}
 function updateWork(work:Work){works=works.some(w=>w.id===work.id)?works.map(w=>w.id===work.id?work:w):[...works,work];}
 async function refreshWorks(){const r=await fetch('/api/v1/discovery/library');if(r.ok)works=await r.json();}
 async function mark(entry:Entry,completed:boolean){if(!permitted(entry,"play")){libraryError="Modification de progression non autorisée.";return}try{if(active?.id===entry.id){closePlayer();await tick();}await waitForProgress(entry.id);const r=await fetch('/api/v1/media/'+entry.id+'/state',{method:'PATCH',headers:{'Content-Type':'application/json','X-Mosaune-Token':status?.token??''},body:JSON.stringify({completed,revision:entry.revision})});if(!r.ok)throw new Error((await r.json()).error);await refreshLibrary();}catch(e){libraryError=e instanceof Error?e.message:'État non enregistré.';}}
 function openResume(card:Card){if(card.entry){const g=groupMedia(entries).find(g=>g.entries.some(e=>e.id===card.entry?.id));if(g){void openCard(groupCard(g));return;}}void openCard(card);}
  let arrivals='';let tracking=false;let arrivalsKnown:Map<string,number>|null=null;
 async function checkArrivals(){if(!status||tracking||document.hidden||!kinds.some(k=>k==='films'||k==='series'))return;tracking=true;try{const list=await acquisition<DownloadRequest[]>(status.token,'requests');const changed=list.filter(r=>r.local>(arrivalsKnown?.get(r.id)??0));if(arrivalsKnown&&changed.length){arrivals=changed.map(r=>`« ${r.title} » : ${r.local} fichier${r.local>1?'s':''} disponible${r.local>1?'s':''}.`).join(' ');await refreshLibrary();}arrivalsKnown=new Map(list.map(r=>[r.id,r.local]));}catch{/* Optional notifications never interrupt playback. */}finally{tracking=false;}}
 let discoveryFocus = '';
  let inspected: Entry | null = null;
  let providers: Provider[] = [];
  type Status = { version: string; stage: string; modules: { id: string; name: string; kinds: Kind[] }[]; libraryConnected: boolean; token: string; user:Account };
  let libraries: { id: string; name: string; path: string; kind: string; count: number }[] = [];
  let startOver = false;
  let playbackKey = 0;
  let libraryError = '';
 let catalogueWarnings:string[]=[];

  let status: Status | null = null;
  let connectionError = false;
  let loading = true;
  let view: View = 'home';
  let query = '';
  let entries: Entry[] = [];
  let active: Entry | null = null;
  let importKind: Kind = 'films';
  let chooser: HTMLInputElement;
  let notice = '';
  let musicQueue = emptyQueue();
  let queueOpen = false;
  let playlistPending: Entry[] | null = null;
  let musicMessage = '';
  const queueStorageKey = 'mosaune.music.queue.v1.'+user.id;
  $: repeatLabel = musicQueue.repeat === 'off' ? 'Répétition : non' : musicQueue.repeat === 'all' ? 'Répéter la file' : 'Répéter ce morceau';
  const labels = { films: 'Films', series: 'Séries', music: 'Musique', watchlist: 'Ma liste', favorites: 'Favoris', playlists:'Playlists', requests:'Demandes', settings:'Réglages', home: 'Accueil' };
  $: kinds = status?.modules.flatMap(m => m.kinds) ?? [];
  $: visible = entries.filter(e => (view === 'home' || e.kind === view) && [e.name,e.metadata?.artist,e.metadata?.album,e.metadata?.albumArtist,e.metadata?.series].filter(Boolean).join(' ').toLocaleLowerCase('fr').includes(query.toLocaleLowerCase('fr')));
  $: sectionTitle = view === 'home' ? 'Votre espace, à votre rythme.' : labels[view];

  async function refreshIdentified(){await refreshLibrary();if(page?.type==='content'&&inspected){const group=groupMedia(entries).find(g=>g.entries.some(e=>e.id===inspected?.id));if(group){page={type:'content',card:groupCard(group)};pageKey++;}}}
  async function refreshProviders(){try{const r=await fetch('/api/v1/metadata/providers');if(!r.ok)return;const next:Provider[]=await r.json();const stamp=(list:Provider[])=>list.map(p=>p.id+':'+p.enabled+':'+p.configured).join('|');if(stamp(next)!==stamp(providers)){providers=next;pageKey++;}}catch{/* Existing state remains usable offline. */}}
  async function connect() {
    loading = true;
    connectionError = false;
    try {
      const response = await fetch('/api/v1/status');
      if (!response.ok) throw new Error('server');
      const data = await response.json() as Status;
      if (data.stage !== 'local-prototype' || !Array.isArray(data.modules)) throw new Error('schema');
      if(data.user.id!==user.id){sessionEnded();return;}
 status = data;
 const pr=await fetch('/api/v1/metadata/providers');if(pr.ok)providers=await pr.json();
      await refreshLibrary();await refreshNavigation();await checkArrivals();
    } catch { connectionError = true; status = null; }
    finally { loading = false; }
  }

  onMount(() => {
    try { musicQueue = restoreQueue(localStorage.getItem(queueStorageKey)); }
    catch { musicMessage = 'Ce navigateur ne permet pas de conserver la file d’écoute.'; }
    void connect();
    void refreshAppearance();
    const extensionTimer=setInterval(()=>{if(!document.hidden){void refreshAppearance();void refreshProviders();void refreshNavigation();}},30000);
    const extensionRefresh=()=>{void refreshAppearance();void refreshProviders();void refreshNavigation();};
    const catalogueRefresh=()=>{page=null;backStack=[];void refreshLibrary().catch(()=>{});};
 window.addEventListener('mosaune:catalogue-changed',catalogueRefresh);
 window.addEventListener('mosaune:extensions-changed',extensionRefresh);
    window.addEventListener('focus',extensionRefresh);
    const arrivalTimer=setInterval(()=>void checkArrivals(),30000);
    return () => {window.removeEventListener('mosaune:catalogue-changed',catalogueRefresh);clearInterval(extensionTimer);window.removeEventListener('mosaune:extensions-changed',extensionRefresh);window.removeEventListener('focus',extensionRefresh);clearInterval(arrivalTimer);entries.filter(e => !e.libraryId).forEach(e => URL.revokeObjectURL(e.url));};
  });

  function navigate(next: View) {externalPage=null;view=next;page=null;backStack=[];query='';notice='';pageKey++;window.scrollTo({top:0});}
  function discoverEntry(entry: Entry) {inspected=null;const group=groupMedia(entries).find(g=>g.entries.some(e=>e.id===entry.id));if(group)void openCard(groupCard(group));}
  async function choose(kind: Kind) {
    importKind = kind;
    await tick();
    chooser.click();
  }
  function addFiles(event: Event) {
    const input = event.currentTarget as HTMLInputElement;
    const files = Array.from(input.files ?? []);
    if (!files.length) return;
    const imported = files.map(file => ({
      id: crypto.randomUUID(), name: file.name.replace(/\.[^.]+$/, ''), filename: file.name,
      kind: importKind, url: URL.createObjectURL(file), size: file.size,
    }));
    entries = [...entries, ...imported];
    view = importKind;
    query = '';
    notice = `${imported.length} fichier${imported.length > 1 ? 's' : ''} ajouté${imported.length > 1 ? 's' : ''} pour cette session. Aucun transfert vers le serveur.`;
    input.value = '';
  }
  async function refreshLibrary() {
    try {
      const response = await fetch('/api/v1/library');
      if (!response.ok) throw new Error('Impossible de charger la bibliothèque.');
      const data = await response.json() as { libraries: typeof libraries; items: Entry[]; catalogueWarnings?:string[] };
      libraries = data.libraries;
 catalogueWarnings=data.catalogueWarnings??[];
      entries = [...data.items, ...entries.filter(e => !e.libraryId)];
      if (inspected) inspected=entries.find(e=>e.id===inspected?.id) ?? null;
      libraryError = ''; await refreshWorks();
    } catch { libraryError = 'La bibliothèque est inaccessible. Vérifiez le serveur puis réessayez.'; throw new Error(libraryError); }
  }
  function play(entry: Entry, restart = false) {
 if(!permitted(entry,"play")){libraryError="Votre compte peut consulter cette bibliothèque, mais n’a pas la permission de lire ses fichiers.";return;}
    if (entry.kind === 'music') {
      const album = entry.libraryId ? groupMedia(entries).find(g => g.entries.some(e => e.id === entry.id))?.entries ?? [entry] : entries.filter(e => !e.libraryId && e.kind === 'music');
      playMusic(album, album.findIndex(e => e.id === entry.id)); if(!restart)startOver=false; return;
    }
    if (active?.id === entry.id && !restart) return;
    startEntry(entry, restart);
  }
  function startEntry(entry: Entry, restart = false) {
 if(!permitted(entry,"play")){musicMessage="Lecture non autorisée pour cette bibliothèque.";return;}
    active = entry; startOver = restart; playbackKey++;
  }
  function setQueue(next: MusicQueue) {
    musicQueue = next;
    try { localStorage.setItem(queueStorageKey, savedQueue(next, entries)); }
    catch { musicMessage = 'La lecture continue, mais la file ne pourra pas être retrouvée après rechargement.'; }
  }
  function playMusic(tracks: Entry[], start = 0) {
    if (!tracks.length || tracks.some(e => e.kind !== 'music')) return;
    try { musicMessage = ''; setQueue(makeQueue(tracks.map(e => e.id), start, musicQueue.repeat)); startEntry(tracks[start], true); }
    catch (e) { musicMessage = e instanceof Error ? e.message : 'Impossible de préparer la file.'; }
  }
  function appendMusic(tracks: Entry[]) {
    try { setQueue(appendQueue(musicQueue, tracks.filter(e => e.kind === 'music').map(e => e.id))); queueOpen = true; }
    catch (e) { musicMessage = e instanceof Error ? e.message : 'Impossible d’ajouter les morceaux.'; }
  }
  function playQueued(item: QueueItem) {
    const entry = entries.find(e => e.id === item.id && e.kind === 'music');
    if (!entry) { musicMessage = 'Ce morceau est indisponible. Actualisez son dossier.'; return; }
    setQueue({ ...musicQueue, current: item.key }); startEntry(entry, true);
  }
  function stepMusic(direction = 1, ended = false) {
    const available = new Set(entries.filter(e => e.kind === 'music').map(e => e.id));
    const item = nextInQueue(musicQueue, available, direction, ended);
    if (item) playQueued(item);
    else if (direction < 0) {
      const current = musicQueue.items.find(item => item.key === musicQueue.current);
      if (current && available.has(current.id)) playQueued(current);
    }
  }
  function removeQueued(key: string) {
    const index = musicQueue.items.findIndex(item => item.key === key);
    const items = musicQueue.items.filter(item => item.key !== key);
    if (musicQueue.current !== key) { setQueue({ ...musicQueue, items }); return; }
    const replacement = [...items.slice(index), ...items.slice(0, index).reverse()].find(item => entries.some(e => e.id === item.id && e.kind === 'music'));
    setQueue({ ...musicQueue, items, current: replacement?.key ?? null });
    if (active?.kind === 'music') { if (replacement) playQueued(replacement); else closePlayer(); }
  }
  function clearQueue() { if (active?.kind === 'music') closePlayer(); setQueue({ ...emptyQueue(), repeat: musicQueue.repeat }); }
  function saveQueue() {
    const tracks = musicQueue.items.map(item => entries.find(e => e.id === item.id));
    if (tracks.some(e => !e)) { musicMessage = 'Retirez les morceaux indisponibles de la file ou actualisez leur dossier avant de créer la playlist.'; return; }
    playlistPending = tracks as Entry[];
  }
  function repeatMusic() { setQueue({ ...musicQueue, repeat: musicQueue.repeat === 'off' ? 'all' : musicQueue.repeat === 'all' ? 'one' : 'off' }); }
  function recordProgress(id: string, position: number, duration: number, completed: boolean) {
    entries = entries.map(e => e.id === id ? { ...e, position, duration, completed,lastPlayed:Date.now()/1000 } : e);
  }
  function timestamp(seconds: number) { const s = Math.floor(seconds); return `${Math.floor(s / 60)}:${String(s % 60).padStart(2, '0')}`; }
  function closePlayer() { active = null; }
  function nextTrack(ended = false) {
    if (!active) return;
    if (active.kind === 'music') { stepMusic(1, ended); return; }
    const queue = active.libraryId ? groupMedia(entries).find(g=>g.entries.some(e=>e.id===active?.id))?.entries ?? [] : entries.filter(e=>!e.libraryId && e.kind===active?.kind);
    const index = queue.findIndex(e => e.id === active?.id);
    if (index >= 0 && index + 1 < queue.length) void play(queue[index + 1]);
  }
  async function remove(entry: Entry) {
    if (active?.id === entry.id) { closePlayer(); await tick(); }
    entries = entries.filter(e => e.id !== entry.id);
    if (entry.kind === 'music') setQueue({ ...musicQueue, items: musicQueue.items.filter(item => item.id !== entry.id), current: musicQueue.items.find(item => item.key === musicQueue.current)?.id === entry.id ? null : musicQueue.current });
    URL.revokeObjectURL(entry.url);
  }
  function size(bytes: number) { return bytes >= 1024 ** 3 ? `${(bytes / 1024 ** 3).toFixed(1)} Go` : `${(bytes / 1024 ** 2).toFixed(1)} Mo`; }
</script>
<svelte:head><title>{externalPage?externalTitle:page?.type==='content'?page.card.title:page?.type==='person'?page.name:labels[view]} · Mosaune</title>{#if appearanceLink(extensionAppearance)}<link rel="stylesheet" href={appearanceLink(extensionAppearance)}/>{/if}</svelte:head>
<div class="app-shell experience-shell" data-extension-theme={extensionAppearance.stamp} data-ui-section={view} class:with-player={active}>
 <aside class="sidebar"><a class="brand" href="#accueil" onclick={()=>navigate('home')} aria-label="Mosaune, accueil"><span class="brand-mark"><span></span><span></span><span></span></span><span>Mosaune<small>LE GOÛT DES HISTOIRES</small></span></a>
 {#each [{id:'main',label:'VOTRE ESPACE',aria:'Navigation principale'},{id:'personal',label:'VOS ENVIES',aria:'Vos listes'}] as group}<div class="nav-caption" class:secondary-caption={group.id==='personal'}>{group.label}</div><nav aria-label={group.aria}>{#each navigation.filter(p=>p.placement===group.id) as item(uiPageKey(item))}<button class:current={item.screen?!externalPage&&view===item.screen:item.owner?externalPage?.owner===item.owner&&externalPage?.id===item.id:!externalPage&&view===item.id} onclick={()=>openUI(item)}><Icon name={item.icon||'home'}/>{item.label}</button>{/each}</nav>{/each}
 <div class="sidebar-bottom"><div class="account-identity"><strong>{user.name}</strong><button class="text-action" onclick={onlogout}>Se déconnecter</button></div><button class="settings-nav" class:current={view==='settings'} onclick={()=>navigate('settings')}><Icon name="settings"/>Réglages</button><div class="server-card"><Icon name="server"/><div><strong>Votre collection, chez vous</strong><span>{connectionError?'Serveur inaccessible':loading?'Connexion…':'Connecté à cet ordinateur'}</span></div></div><div class="version">PROJET MOSAUNE <span>v{status?.version??'0.0.45'}</span></div></div></aside>
 <main id="main" class:settings-main={view==='settings'}><header class="topbar experience-topbar"><div class="page-navigation">{#if page}<button class="icon-button" aria-label="Revenir à la page précédente" onclick={back}>←</button>{/if}<span>{externalPage?externalTitle:labels[view]}</span></div>{#if status}<SearchBox {works} {entries} {providers} {kinds} context={view} token={status.token} open={openCard}/>{/if}{#if kinds.includes('music')}<button class="queue-toggle" aria-label="Ouvrir la file d’écoute" aria-expanded={queueOpen} onclick={()=>queueOpen=!queueOpen}><Icon name="queue"/><span>{musicQueue.items.length}</span></button>{/if}</header>
 {#if connectionError}<div class="connection-warning" role="alert">Le serveur ne répond pas. <button onclick={connect}>Réessayer</button></div>{/if}
 {#each catalogueWarnings as warning}<p class="extension-notice" role="status">{warning}</p>{/each}
 {#if libraryError}<p class="playback-error" role="alert">{libraryError}</p>{/if}
 <div hidden={view!=='settings'} class="settings-page"><div class="page-heading"><span class="eyebrow">FAITES COMME CHEZ VOUS</span><h1>Paramètres</h1><p>Un espace pour chaque réglage de Mosaune.</p></div>
 <div class="settings-layout"><nav class="settings-navigation" aria-label="Rubriques des paramètres">{#each settingsSections as section}<button aria-current={settingsTab===uiPageKey(section)?'page':undefined} class:selected={settingsTab===uiPageKey(section)} onclick={()=>{settingsTab=uiPageKey(section);document.querySelector('.settings-content')?.scrollTo({top:0});}}><strong>{section.label}</strong><small>{section.description}</small></button>{/each}</nav>
 <div class="settings-content">{#if status}{#if settingsPage?.owner}<ExtensionPage owner={settingsPage.owner} id={settingsPage.id} {entries} {works} {kinds} {providers} token={status.token} {play} {mark} open={openResume} {playAlbum} navigate={uiNavigate} revision={uiRevision}/>{/if}
 {#if settingsTab==='extensions'}<ExtensionSettings {user} token={status.token} refresh={refreshLibrary} openSettings={id=>{settingsTab=id;document.querySelector('.settings-content')?.scrollTo({top:0});}}/>{/if}
 {#if settingsTab==='account'}<AccountSettings {user} token={status.token} changed={accountChanged} logout={onlogout}/>{/if}
 {#if user.admin&&settingsTab==='users'}<AccountSettings admin {user} token={status.token} {libraries} changed={accountChanged} logout={sessionEnded}/>{/if}
 {#if user.admin}<div hidden={settingsTab!=='libraries'}><LibraryManager {libraries} {kinds} token={status.token} refresh={refreshLibrary}/></div>
 <div hidden={settingsTab!=='metadata'}><MetadataSettings token={status.token} refresh={refreshLibrary} bind:providers/>{#if kinds.includes('films')||kinds.includes('series')}<MetadataQueue token={status.token} refresh={refreshLibrary}/>{/if}</div>
 {#if kinds.includes('films')||kinds.includes('series')}<div hidden={settingsTab!=='downloads'}><DownloadSettings token={status.token} {libraries} refresh={refreshLibrary}/></div>{/if}
 <div hidden={settingsTab!=='server'}><section class="settings-server"><span class="eyebrow">VOTRE INSTALLATION</span><h2>Serveur Mosaune</h2><dl><dt>Version</dt><dd>{status.version}</dd><dt>Accès</dt><dd>Local · cet ordinateur</dd><dt>Modules actifs</dt><dd>{status.modules.map(m=>m.name).join(', ')||'Aucun'}</dd></dl><p>Gardez la fenêtre du serveur ouverte pendant l’utilisation. Avant une mise à jour, arrêtez Mosaune et faites une copie du dossier data.</p></section><section class="settings-extras"><h2>Essai ponctuel</h2><p>Ouvrir des fichiers pour cette session, sans les ajouter au catalogue.</p>{#each kinds as kind}<button class="secondary" onclick={()=>choose(kind)}>Ouvrir · {labels[kind]}</button>{/each}</section></div>
 {/if}{/if}</div></div></div>
 {#if status&&kinds.includes('music')}<PlaylistManager show={view==='playlists'} token={status.token} {entries} bind:pending={playlistPending} play={playMusic} append={appendMusic}/>{/if}
 {#if status&&view==='requests'}<DownloadRequests token={status.token} refresh={refreshLibrary} open={openCard} admin={user.admin} settings={()=>{settingsTab='downloads';navigate('settings');}}/>{/if}
 {#if status&&externalPage}<ExtensionPage owner={externalPage.owner} id={externalPage.id} {entries} {works} {kinds} {providers} token={status.token} {play} {mark} open={openResume} {playAlbum} navigate={uiNavigate} revision={uiRevision} titleChanged={title=>externalTitle=title}/>{/if}
 {#if status&&!loading&&!externalPage&&view!=='settings'&&view!=='playlists'&&view!=='requests'}{#key pageKey}{#if page?.type==='content'}<ContentPage card={page.card} {entries} {works} {providers} token={status.token} {play} enqueue={appendMusic} playlist={tracks=>playlistPending=tracks} inspect={entry=>{if(permitted(entry,"edit"))inspected=entry;else libraryError="Vous n’avez pas la permission de modifier cette fiche."}} open={openCard} person={openPerson} artist={openArtist} changed={updateWork} refresh={refreshLibrary} {mark}/>{:else if page?.type==='person'}<PersonPage provider={page.provider} id={page.id} name={page.name} {entries} token={status.token} open={openCard}/>{:else}{#if activeScreen?.owner}<ExtensionPage owner={activeScreen.owner} id={activeScreen.id} {entries} {works} {kinds} {providers} token={status.token} open={openResume} {playAlbum} {play} {mark} navigate={uiNavigate} revision={uiRevision}/>{:else}<div class="quiet-empty"><h1>Cette page n’est pas activée.</h1><p>Activez une extension d’interface pour afficher cette rubrique. Vos médias et leurs fiches sont conservés.</p><button class="secondary" onclick={()=>{navigate('settings');settingsTab='extensions';}}>Ouvrir les extensions</button></div>{/if}{/if}{/key}{/if}
 <footer><span>Projet Mosaune</span><span>Le cinéma. Les albums. Et toutes vos envies.</span></footer>
 {#if providers.length}<details class="credits"><summary>Sources des informations</summary>{#each providers as p}<p><a href={p.website} target="_blank" rel="noreferrer">{p.name}</a> — {p.attribution}</p>{#if p.id==='tmdb'}<img class="tmdb-logo" src="/tmdb-logo.svg" alt="TMDB"/>{/if}{/each}</details>{/if}
 </main>
  {#if active}
    <section class:video-player={active.kind !== 'music'} class="player" aria-label="Lecteur">
      <div class="player-header"><div class="player-label"><Icon name={active.kind} size={22} /><div><strong>{active.name}</strong><span>{active.kind === 'music' ? active.metadata?.artist ?? 'Écoute locale' : 'Vidéo locale · contrôles du navigateur'}</span></div></div><div class="player-actions">{#if active.kind === 'music'}<button class="icon-button" aria-label="Morceau précédent" onclick={() => stepMusic(-1)}><Icon name="previous" /></button>{/if}<button class="icon-button" aria-label="Fichier suivant" onclick={() => nextTrack()}><Icon name="next" /></button><button class="icon-button" aria-label="Fermer le lecteur" onclick={closePlayer}><Icon name="close" /></button></div></div>
      {#if active.kind === 'music'}<div class="music-player-controls"><button onclick={() => setQueue(shuffleQueue(musicQueue))}>Mélanger la suite</button><button class:enabled={musicQueue.repeat !== 'off'} onclick={repeatMusic}>{repeatLabel}</button><button aria-expanded={queueOpen} onclick={() => queueOpen = !queueOpen}>File · {musicQueue.items.length}</button></div>{/if}
      {#key playbackKey}<MediaPlayer entry={active} token={status?.token ?? ''} {startOver} next={() => nextTrack(true)} progress={recordProgress} />{/key}
    </section>
  {/if}
</div>

{#if arrivals}<div class="arrival-message" role="status">{arrivals}<button class="text-action" onclick={()=>{arrivals='';navigate('requests');}}>Voir les demandes</button><button class="text-action" aria-label="Fermer la notification" onclick={()=>arrivals=''}>Fermer</button></div>{/if}
{#if musicMessage}<div class="music-message" role="status">{musicMessage}<button class="restart-button" aria-label="Fermer le message musical" onclick={() => musicMessage = ''}>Fermer</button></div>{/if}
{#if queueOpen && kinds.includes('music')}<QueuePanel queue={musicQueue} {entries} activeKey={active?.kind === 'music' ? musicQueue.current : null} play={playQueued} move={(from,to) => setQueue({ ...musicQueue, items: moveItem(musicQueue.items, from, to) })} remove={removeQueued} clear={clearQueue} shuffle={() => setQueue(shuffleQueue(musicQueue))} save={saveQueue} close={() => queueOpen = false}/>{/if}

{#if inspected && status}<MetadataDetails entry={entries.find(e=>e.id===inspected?.id)||inspected} {providers} token={status.token} refresh={refreshIdentified} discover={discoverEntry} close={()=>inspected=null} />{/if}

<input class="file-input" aria-label="Choisir les fichiers locaux" bind:this={chooser} type="file" multiple accept={importKind === 'music' ? 'audio/*,.flac,.m4a,.m4b,.opus' : 'video/*,.mkv,.mp4,.webm,.mov'} onchange={addFiles} />
