<script>
    import { createEventDispatcher } from "svelte";

    export let showModal;
    export let currentClockSpeed;
    export let currentDisplayColor;
    export let currentScanlineEffect;
    export let currentKeyMap;

    const dispatch = createEventDispatcher();

    // --- Tab State ---
    let activeTab = 'display'; // 'display', 'controls', or 'performance'

    // --- Local State for Settings ---
    let newClockSpeed = currentClockSpeed;
    let newDisplayColor = currentDisplayColor;
    let newScanlineEffect = currentScanlineEffect;
    let newKeyMap = { ...currentKeyMap };

    let remappingKey = null; // Stores the CHIP-8 key being remapped

    function closeModal() {
        showModal = false;
    }

    function saveSettings() {
        dispatch("save", {
            clockSpeed: newClockSpeed,
            displayColor: newDisplayColor,
            scanlineEffect: newScanlineEffect,
            keyMap: newKeyMap,
        });
        closeModal();
    }

    // --- Key Remapping Logic (no changes needed here) ---
    function startRemap(event, chip8Key) {
        remappingKey = chip8Key;
        event.target.value = "Press key...";
        window.addEventListener("keydown", handleRemapKeyDown, { once: true });
    }

    function handleRemapKeyDown(event) {
        event.preventDefault();
        if (remappingKey !== null) {
            newKeyMap[remappingKey] = event.key.toLowerCase();
            // Manually trigger reactivity for the input field
            newKeyMap = newKeyMap;
            remappingKey = null;
        }
    }

    function endRemap(event, chip8Key) {
        if (remappingKey !== null) {
            // If user blurs without pressing a key, revert
            event.target.value = currentKeyMap[chip8Key];
            remappingKey = null;
        }
    }
</script>

{#if showModal}
    <div class="fixed inset-0 bg-black bg-opacity-60 flex items-center justify-center z-50 transition-opacity">
        <div class="bg-[#34495e] p-6 rounded-lg shadow-xl border border-gray-700 w-full max-w-lg">
            <h2 class="text-2xl font-bold mb-4 text-center text-cyan-400">Settings</h2>

            <!-- Tab Navigation -->
            <div class="border-b border-gray-600 mb-4">
                <nav class="-mb-px flex space-x-4" aria-label="Tabs">
                    <button
                        on:click={() => activeTab = 'display'}
                        class="px-3 py-2 font-medium text-sm rounded-t-md"
                        class:text-cyan-400={activeTab === 'display'}
                        class:border-cyan-400={activeTab === 'display'}
                        class:border-b-2={activeTab === 'display'}
                        class:text-gray-400={activeTab !== 'display'}
                        class:hover:text-white={activeTab !== 'display'}
                        class:border-transparent={activeTab !== 'display'}>
                        Display
                    </button>
                    <button
                        on:click={() => activeTab = 'controls'}
                        class="px-3 py-2 font-medium text-sm rounded-t-md"
                        class:text-cyan-400={activeTab === 'controls'}
                        class:border-cyan-400={activeTab === 'controls'}
                        class:border-b-2={activeTab === 'controls'}
                        class:text-gray-400={activeTab !== 'controls'}
                        class:hover:text-white={activeTab !== 'controls'}
                        class:border-transparent={activeTab !== 'controls'}>
                        Controls
                    </button>
                     <button
                        on:click={() => activeTab = 'performance'}
                        class="px-3 py-2 font-medium text-sm rounded-t-md"
                        class:text-cyan-400={activeTab === 'performance'}
                        class:border-cyan-400={activeTab === 'performance'}
                        class:border-b-2={activeTab === 'performance'}
                        class:text-gray-400={activeTab !== 'performance'}
                        class:hover:text-white={activeTab !== 'performance'}
                        class:border-transparent={activeTab !== 'performance'}>
                        Performance
                    </button>
                </nav>
            </div>

            <!-- Tab Content -->
            <div class="min-h-[280px]">
                {#if activeTab === 'display'}
                    <!-- Display Settings -->
                    <div class="space-y-4">
                        <div>
                            <label class="block text-gray-300 text-sm font-bold mb-2">Display Color</label>
                            <div class="flex flex-wrap gap-4">
                                <label class="inline-flex items-center"><input type="radio" class="form-radio" name="displayColor" value="#33FF00" bind:group={newDisplayColor} /><span class="ml-2">Classic Green</span></label>
                                <label class="inline-flex items-center"><input type="radio" class="form-radio" name="displayColor" value="#FFFFFF" bind:group={newDisplayColor} /><span class="ml-2">White</span></label>
                                <label class="inline-flex items-center"><input type="radio" class="form-radio" name="displayColor" value="#FFBF00" bind:group={newDisplayColor} /><span class="ml-2">Amber</span></label>
                            </div>
                        </div>
                        <div>
                            <label class="inline-flex items-center"><input type="checkbox" class="form-checkbox" bind:checked={newScanlineEffect} /><span class="ml-2 text-gray-300">Enable Scanline Effect</span></label>
                        </div>
                    </div>
                {/if}

                {#if activeTab === 'controls'}
                    <!-- Controls/Key Remapping -->
                    <div>
                        <h3 class="text-lg font-bold mb-2 text-gray-200">Key Remapping</h3>
                        <p class="text-sm text-gray-400 mb-4">Click an input, then press the desired keyboard key.</p>
                        <div class="grid grid-cols-4 gap-2 text-center">
                            {#each Object.entries(newKeyMap).sort((a,b) => parseInt(a[0], 16) - parseInt(b[0], 16)) as [chip8Key, keyboardKey]}
                                <div class="bg-gray-700 p-2 rounded-md border border-gray-600">
                                    <span class="font-bold text-gray-300">{parseInt(chip8Key, 16).toString(16).toUpperCase()}</span>
                                    <input
                                        type="text"
                                        value={keyboardKey}
                                        on:focus={(e) => startRemap(e, chip8Key)}
                                        on:blur={(e) => endRemap(e, chip8Key)}
                                        class="w-full bg-gray-600 text-white text-center rounded-sm mt-1 focus:outline-none focus:ring-2 focus:ring-blue-500 p-1"
                                        readonly
                                    />
                                </div>
                            {/each}
                        </div>
                    </div>
                {/if}

                 {#if activeTab === 'performance'}
                    <!-- Performance Settings -->
                    <div class="space-y-6">
                        <div>
                             <label for="clockSpeed" class="block text-gray-300 text-sm font-bold mb-2">CPU Clock Speed: {newClockSpeed} Hz</label>
                            <input type="range" id="clockSpeed" min="100" max="2000" step="50" bind:value={newClockSpeed} class="w-full h-2 bg-gray-200 rounded-lg appearance-none cursor-pointer dark:bg-gray-700" />
                        </div>
                        <div>
                             <label class="block text-gray-300 text-sm font-bold mb-2">Speed Presets</label>
                            <div class="flex flex-wrap gap-4">
                               <label class="inline-flex items-center"><input type="radio" class="form-radio" value={700} bind:group={newClockSpeed} /><span class="ml-2">Original (700Hz)</span></label>
                               <label class="inline-flex items-center"><input type="radio" class="form-radio" value={1400} bind:group={newClockSpeed} /><span class="ml-2">Fast (1400Hz)</span></label>
                               <label class="inline-flex items-center"><input type="radio" class="form-radio" value={2000} bind:group={newClockSpeed} /><span class="ml-2">Turbo (2000Hz)</span></label>
                           </div>
                        </div>
                    </div>
                {/if}
            </div>

            <!-- Action Buttons -->
            <div class="flex justify-end gap-3 mt-6 border-t border-gray-600 pt-4">
                <button on:click={closeModal} class="bg-gray-600 hover:bg-gray-700 text-white font-bold py-2 px-4 rounded-md transition-colors">Cancel</button>
                <button on:click={saveSettings} class="bg-green-600 hover:bg-green-700 text-white font-bold py-2 px-4 rounded-md transition-colors">Save & Close</button>
            </div>
        </div>
    </div>
{/if}
