<script>
    import { onMount, onDestroy } from 'svelte';
    import { GetLogs } from '../wailsjs/go/main/App';

    let logs = [];
    let intervalId;
    let logViewerElement;

    async function fetchLogs() {
        logs = await GetLogs();
        // Scroll to bottom on new logs
        if (logViewerElement) {
            logViewerElement.scrollTop = logViewerElement.scrollHeight;
        }
    }

    onMount(() => {
        fetchLogs();
        intervalId = setInterval(fetchLogs, 500); // Fetch logs every 500ms
    });

    onDestroy(() => {
        clearInterval(intervalId);
    });
</script>

<div class="bg-slate-800 p-2 rounded-md border border-slate-700 font-mono text-xs overflow-y-scroll h-64" bind:this={logViewerElement}>
    {#each logs as log}
        <div>{log}</div>
    {/each}
</div>

<style>
    /* Add any specific styles for the log viewer here */
</style>
