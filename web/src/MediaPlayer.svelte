<script lang="ts">
  import { onMount } from 'svelte';
  import { queueProgress } from './progress';
  export let entry: { id: string; name: string; kind: string; url: string; libraryId?: string; revision?: string; position?: number; completed?: boolean };
  export let token: string;
  export let startOver = false;
  export let next: () => void;
  export let progress: (id: string, position: number, duration: number, completed: boolean) => void;
  let element: HTMLMediaElement;
  let ready = false;
  let error = '';
  let saveError = '';
  let lastSave = 0;
  let alive = true;

  function save(force = false, completed = false) {
    if (!ready || !entry.libraryId || !element || !Number.isFinite(element.duration) || element.duration <= 0) return;
    if (!force && Date.now() - lastSave < 5000) return;
    lastSave = Date.now();
    const position = Math.min(element.currentTime, element.duration);
    const duration = element.duration;
    const id = entry.id;
    const body = JSON.stringify({ position, duration, completed, revision: entry.revision });
    progress(id, position, duration, completed);
    queueProgress(id, { body, token, result: saved => {
      if (alive) saveError = saved ? '' : 'La progression n’a pas été enregistrée. Vérifiez que le serveur est ouvert.';
    } });
  }
  async function loaded() {
    const resume = !startOver && !entry.completed ? entry.position ?? 0 : 0;
    if (resume > 0 && Number.isFinite(element.duration)) element.currentTime = Math.min(resume, Math.max(0, element.duration - 1));
    ready = true;
    try { await element.play(); }
    catch (e) { if (e instanceof DOMException && e.name === 'NotAllowedError') error = 'Appuyez sur lecture pour démarrer.'; else if (!(e instanceof DOMException && e.name === 'AbortError')) unsupported(); }
  }
  function unsupported() { error = 'Ce fichier ou une de ses pistes n’est pas pris en charge par le navigateur. La conversion audio/vidéo arrivera dans une prochaine étape.'; }
  function ended() { save(true, true); next(); }
  onMount(() => {
    const flush = () => save(true, element?.ended ?? false);
    const hidden = () => { if (document.visibilityState === 'hidden') flush(); };
    window.addEventListener('pagehide', flush);
    document.addEventListener('visibilitychange', hidden);
    return () => { flush(); alive = false; element?.pause(); window.removeEventListener('pagehide', flush); document.removeEventListener('visibilitychange', hidden); };
  });
</script>

{#if entry.kind === 'music'}
  <audio bind:this={element} src={entry.url} controls preload="metadata" onloadedmetadata={loaded} ontimeupdate={() => save(false, element.ended)} onpause={() => save(true, element.ended)} onseeked={() => save(true, element.ended)} onended={ended} onerror={unsupported} aria-label={`Lecture de ${entry.name}`}></audio>
{:else}
  <video bind:this={element} src={entry.url} controls preload="metadata" playsinline onloadedmetadata={loaded} ontimeupdate={() => save(false, element.ended)} onpause={() => save(true, element.ended)} onseeked={() => save(true, element.ended)} onended={ended} onerror={unsupported} aria-label={`Lecture de ${entry.name}`}><track kind="captions" /></video>
{/if}
{#if error}<p class="playback-error" role="alert">{error}</p>{/if}
{#if saveError}<p class="playback-error" role="alert">{saveError}</p>{/if}
