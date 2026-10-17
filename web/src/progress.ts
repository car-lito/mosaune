type Update = { body: string; token: string; result: (saved: boolean) => void };
type Queue = { latest?: Update; done?: (()=>void)[] };
const queues = new Map<string, Queue>();

// One request and at most one newer snapshot per video, even with a slow server.
// Shared between player instances so restarting the same video preserves write order.
export function queueProgress(id: string, update: Update) {
  const existing = queues.get(id);
  if (existing) { existing.latest = update; return; }
  const queue: Queue = { latest: update };
  queues.set(id, queue);
  void drain(id, queue);
}

async function drain(id: string, queue: Queue) {
  while (queue.latest) {
    const update = queue.latest;
    queue.latest = undefined;
    try {
      const response = await fetch(`/api/v1/media/${id}/progress`, {
        method: 'PUT', headers: { 'Content-Type': 'application/json', 'X-Mosaune-Token': update.token },
        body: update.body, keepalive: true, signal: AbortSignal.timeout(10000),
      });
      update.result(response.ok);
    } catch { update.result(false); }
  }
  queues.delete(id);queue.done?.forEach(resolve=>resolve());
}

export function waitForProgress(id:string):Promise<void>{const queue=queues.get(id);return queue?new Promise(resolve=>(queue.done??=[]).push(resolve)):Promise.resolve();}
