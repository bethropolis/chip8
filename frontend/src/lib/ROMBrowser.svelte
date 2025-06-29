<script>
    import { onMount } from 'svelte';
    import { GetROMs, LoadROM } from '../wailsjs/go/main/App';
    import { showNotification } from './stores.js';
    import { Play } from 'lucide-svelte';

    let roms = [];
    let selectedROM = '';

    async function fetchROMs() {
        try {
            // FIX: Add a fallback to an empty array in case the backend returns null
            const result = await GetROMs();
            roms = result || [];

            if (roms.length === 0) {
                // This check will now work safely
                showNotification("No ROMs found in ./roms directory.", "warning");
            }
        } catch (error) {
            showNotification(`Failed to load ROM list: ${error}`, "error");
            console.error("Error fetching ROMs:", error);
            roms = []; // Also set to empty array on error to prevent #each from failing
        }
    }

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

    onMount(fetchROMs);
</script>

<!-- The HTML part remains the same. The #each block will now be safe. -->
<div class="bg-gray-800 p-4 rounded-lg shadow-md">
    <h3 class="text-xl font-semibold mb-3 text-center text-cyan-400">ROM Browser</h3>
    <div class="mb-4">
        <select bind:value={selectedROM} class="w-full p-2 rounded-md bg-gray-700 border border-gray-600 text-gray-200">
            <option value="">-- Select a ROM --</option>
            {#each roms as rom}
                <option value={rom}>{rom}</option>
            {/each}
        </select>
    </div>
    <button
        on:click={handleLoadSelectedROM}
        class="w-full flex items-center justify-center space-x-2 bg-blue-600 hover:bg-blue-700 text-white font-medium py-2 px-4 rounded-md transition-colors duration-200"
    >
        <Play size={18} />
        <span>Load Selected ROM</span>
    </button>
</div>
