<script>
    import { onMount } from 'svelte';
    import { GetROMs, LoadROM } from '../wailsjs/go/main/App';
    import { showNotification } from './stores.js';
    import { Play } from 'lucide-svelte';
    // NEW: Import the event listener
    import { EventsOn } from '../wailsjs/runtime/runtime.js';

    let roms = [];
    let selectedROM = '';

    async function fetchROMs() {
        try {
            const result = await GetROMs();
            roms = result || [];
            if (roms.length === 0) {
                showNotification("No ROMs found in the configured directory.", "warning");
            }
        } catch (error) {
            showNotification(`Failed to load ROM list: ${error}`, "error");
            console.error("Error fetching ROMs:", error);
            roms = [];
        }
    }

    // ... (handleLoadSelectedROM is unchanged) ...
    async function handleLoadSelectedROM() {
        if (selectedROM) {
            try {
                await LoadROM(selectedROM);
                showNotification(`ROM loaded: ${selectedROM}`, "success");
            } catch (error) {
                showNotification(`Failed to load ROM: ${error}`, "error");
            }
        } else {
            showNotification("Please select a ROM first.", "warning");
        }
    }

    onMount(() => {
        fetchROMs();
        // NEW: Listen for the backend event to refresh the list
        EventsOn("roms:path-changed", fetchROMs);
    });
</script>

<!-- (The HTML part of this component is unchanged) -->
<style>
    select {
        -webkit-appearance: none; -moz-appearance: none; appearance: none;
        background-image: url('data:image/svg+xml;charset=US-ASCII,%3Csvg%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%20width%3D%22292.4%22%20height%3D%22292.4%22%3E%3Cpath%20fill%3D%22%23CBD5E0%22%20d%3D%22M287%2069.4a17.6%2017.6%200%200%200-13-5.4H18.4c-5%200-9.3%201.8-12.9%205.4A17.6%2017.6%200%200%200%200%2082.2c0%205%201.8%209.3%205.4%2012.9l128%20127.9c3.6%203.6%207.8%205.4%2012.8%205.4s9.2-1.8%2012.8-5.4L287%2095c3.5-3.5%205.4-7.8%205.4-12.8%200-5-1.9-9.2-5.5-12.8z%22%2F%3E%3C%2Fsvg%3E');
        background-repeat: no-repeat;
        background-position: right 0.7rem center;
        background-size: 0.65em auto;
        padding-right: 2.5rem;
    }
</style>
<div class="bg-gray-900 p-3 rounded-md shadow-inner">
    <h3 class="text-lg font-semibold mb-2 text-center text-gray-400">ROM Browser</h3>
    <div class="mb-2">
        <select bind:value={selectedROM} class="w-full p-2 rounded-md bg-gray-700 border border-gray-600 text-gray-200 focus:outline-none focus:ring-2 focus:ring-blue-500">
            <option value="">-- Select a ROM --</option>
            {#each roms as rom}
                <option value={rom}>{rom}</option>
            {/each}
        </select>
    </div>
    <button on:click={handleLoadSelectedROM} class="w-full flex items-center justify-center space-x-2 bg-blue-600 hover:bg-blue-700 text-white font-medium py-2 px-3 rounded-md transition-colors duration-200 text-sm" title="Load Selected ROM">
        <Play size={16} />
        <span>Load ROM</span>
    </button>
</div>
