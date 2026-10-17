<script lang="ts">
  export let name = 'play';
  export let size = 20;
  const paths: Record<string, string> = {
    bookmark: 'M6 3h12v19l-6-4-6 4Z',
 heart: 'M12 21 3 12C-2 4 7-1 12 6c5-7 14-2 9 6Z',
 queue: 'M3 5h18M3 11h12M3 17h8m6-4 5 4-5 4Z',
 settings: 'M12 8a4 4 0 1 0 0 8 4 4 0 0 0 0-8Zm-1-6h2l1 3 3 1 3-1 1 2-2 3v3l2 3-1 2-3-1-3 1-1 3h-2l-1-3-3-1-3 1-1-2 2-3v-3L3 7l1-2 3 1 3-1Z',
 home: 'm3 10 9-7 9 7v10a1 1 0 0 1-1 1h-5v-7H9v7H4a1 1 0 0 1-1-1Z',
    films: 'M3 4h18v16H3ZM7 4v16M17 4v16M3 8h4M3 16h4M17 8h4M17 16h4',
    series: 'M3 6h18v13H3Zm5 16h8M9 2l3 4 3-4',
    music: 'M9 18V5l11-2v13M9 9l11-2M9 18c0 2-2 3-4 3s-3-1-3-2 2-3 4-3 3 1 3 2Zm11-2c0 2-2 3-4 3s-3-1-3-2 2-3 4-3 3 1 3 2Z',
    search: 'M21 21l-5-5M18 10a8 8 0 1 1-16 0 8 8 0 0 1 16 0Z',
    plus: 'M12 5v14M5 12h14',
    arrow: 'M5 12h14m-5-5 5 5-5 5',
    play: 'm8 4 12 8-12 8Z',
    close: 'm6 6 12 12M6 18 18 6',
    server: 'M3 3h18v7H3Zm0 11h18v7H3ZM7 6h.01M7 17h.01M11 6h6M11 17h6',
    check: 'm5 12 4 4L19 6',
    next: 'm5 5 10 7-10 7ZM19 5v14',
    previous: 'm19 5-10 7 10 7ZM5 5v14',
  };
</script>

<svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d={paths[name] ?? paths.play} /></svg>
