<script>
    // NEW: Import the directory selection function
    import { SelectRomsDirectory } from "../wailsjs/go/main/App.js";
    import { settings, updateAndSaveSettings, showNotification } from "./stores.js";
    import { writable } from "svelte/store";

    export let showModal;
    let activeTab = "appearance";
    let localSettings = writable({});

    $: {
        if (showModal) {
            localSettings.set(JSON.parse(JSON.stringify($settings)));
        }
    }

    // ... (keybinding logic is unchanged) ...
    let remappingKey = null;
    let keybindings = [];
    $: {
        if ($localSettings.keyMap) {
            keybindings = Array.from({ length: 16 }, (_, i) => {
                const chip8Hex = i;
                const keyboardKey = Object.keys($localSettings.keyMap).find(
                    k => $localSettings.keyMap[k] === chip8Hex
                ) || "N/A";
                return { chip8Key: chip8Hex, keyboardKey: keyboardKey };
            });
        }
    }
    function closeModal() {
        showModal = false;
    }
    async function saveSettings() {
        await updateAndSaveSettings($localSettings);
        closeModal();
    }
    function startRemap(event, chip8KeyToRemap) {
        remappingKey = chip8KeyToRemap;
        event.target.value = "Press key...";
        window.addEventListener("keydown", handleRemapKeyDown, { once: true });
    }
    function handleRemapKeyDown(event) {
        event.preventDefault();
        if (remappingKey === null) return;
        const newKeyboardKey = event.key.toLowerCase();
        localSettings.update(currentSettings => {
            const updatedKeyMap = { ...currentSettings.keyMap };
            let oldKeyboardKey = Object.keys(updatedKeyMap).find(
                (k) => updatedKeyMap[k] === remappingKey,
            );
            const conflictingChip8Key = updatedKeyMap[newKeyboardKey];
            if (oldKeyboardKey) {
                delete updatedKeyMap[oldKeyboardKey];
            }
            if (conflictingChip8Key !== undefined && oldKeyboardKey) {
                updatedKeyMap[oldKeyboardKey] = conflictingChip8Key;
            } else if (conflictingChip8Key !== undefined) {
                delete updatedKeyMap[newKeyboardKey];
            }
            updatedKeyMap[newKeyboardKey] = remappingKey;
            currentSettings.keyMap = updatedKeyMap;
            return currentSettings;
        });
        remappingKey = null;
    }
    function endRemap(event, chip8Key) {
        if (remappingKey !== null) {
            const originalKey = Object.keys($localSettings.keyMap).find(k => $localSettings.keyMap[k] === chip8Key) || 'N/A';
            event.target.value = originalKey.toUpperCase();
            remappingKey = null;
        }
    }

    // NEW: Function to handle ROM directory selection
    async function browseForRomsPath() {
        try {
            const newPath = await SelectRomsDirectory();
            if (newPath) {
                localSettings.update(s => ({ ...s, romsPath: newPath }));
            }
        } catch (error) {
            showNotification(`Could not select directory: ${error}`, "error");
        }
    }
</script>

{#if showModal}
    <div class="fixed inset-0 bg-black bg-opacity-70 flex items-center justify-center z-50 transition-opacity">
        <div class="bg-gray-800 p-5 rounded-lg shadow-2xl border border-gray-700 w-full max-w-2xl">
            <!-- ... (Header and Tabs are unchanged) ... -->
            <h2 class="text-xl font-semibold mb-4 text-center text-gray-200">Settings</h2>
            <div class="flex space-x-1">
                <div class="w-1/4 bg-gray-900 p-3 rounded-l-md">
                    <nav class="space-y-1">
                        <button on:click={() => (activeTab = "appearance")} class="w-full text-left px-3 py-2 rounded-md text-sm font-medium transition-colors duration-150" class:bg-gray-700={activeTab === "appearance"} class:text-white={activeTab === "appearance"} class:text-gray-400={activeTab !== "appearance"} class:hover:bg-gray-700={activeTab !== "appearance"} class:hover:text-white={activeTab !== "appearance"}>Appearance</button>
                        <button on:click={() => (activeTab = "emulation")} class="w-full text-left px-3 py-2 rounded-md text-sm font-medium transition-colors duration-150" class:bg-gray-700={activeTab === "emulation"} class:text-white={activeTab === "emulation"} class:text-gray-400={activeTab !== "emulation"} class:hover:bg-gray-700={activeTab !== "emulation"} class:hover:text-white={activeTab !== "emulation"}>Emulation</button>
                        <button on:click={() => (activeTab = "keybindings")} class="w-full text-left px-3 py-2 rounded-md text-sm font-medium transition-colors duration-150" class:bg-gray-700={activeTab === "keybindings"} class:text-white={activeTab === "keybindings"} class:text-gray-400={activeTab !== "keybindings"} class:hover:bg-gray-700={activeTab !== "keybindings"} class:hover:text-white={activeTab !== "keybindings"}>Keybindings</button>
                    </nav>
                </div>
                <div class="w-3/4 bg-gray-800 p-4 rounded-r-md">
                    <div class="min-h-[300px]">
                        {#if activeTab === "appearance"}
                            <!-- MODIFIED: Add Pixel Scale setting -->
                            <div class="space-y-5">
                                <h3 class="text-lg font-semibold text-gray-300">Display</h3>
                                <!-- Pixel Color (unchanged) -->
                                <div>
                                    <label class="block text-gray-400 text-sm font-medium mb-2">Pixel Color</label>
                                    <div class="flex flex-wrap gap-4">
                                        <label class="inline-flex items-center"><input type="radio" class="form-radio bg-gray-700 border-gray-600 text-green-500 focus:ring-green-500" name="displayColor" value="#33FF00" bind:group={$localSettings.displayColor} /><span class="ml-2">Classic Green</span></label>
                                        <label class="inline-flex items-center"><input type="radio" class="form-radio bg-gray-700 border-gray-600 text-blue-500 focus:ring-blue-500" name="displayColor" value="#FFFFFF" bind:group={$localSettings.displayColor} /><span class="ml-2">White</span></label>
                                        <label class="inline-flex items-center"><input type="radio" class="form-radio bg-gray-700 border-gray-600 text-yellow-500 focus:ring-yellow-500" name="displayColor" value="#FFBF00" bind:group={$localSettings.displayColor} /><span class="ml-2">Amber</span></label>
                                    </div>
                                </div>
                                <!-- NEW: Pixel Scale setting -->
                                <div>
                                    <label class="block text-gray-400 text-sm font-medium mb-2">Pixel Scale</label>
                                    <div class="flex flex-wrap gap-4">
                                        <label class="inline-flex items-center"><input type="radio" class="form-radio" value={10} bind:group={$localSettings.pixelScale} /><span class="ml-2">10x (Default)</span></label>
                                        <label class="inline-flex items-center"><input type="radio" class="form-radio" value={15} bind:group={$localSettings.pixelScale} /><span class="ml-2">15x</span></label>
                                        <label class="inline-flex items-center"><input type="radio" class="form-radio" value={20} bind:group={$localSettings.pixelScale} /><span class="ml-2">20x</span></label>
                                    </div>
                                </div>
                                <!-- Scanline Effect (unchanged) -->
                                <div>
                                    <label class="inline-flex items-center"><input type="checkbox" class="form-checkbox bg-gray-700 border-gray-600 text-blue-500 focus:ring-blue-500" bind:checked={$localSettings.scanlineEffect} /><span class="ml-2 text-gray-300">Enable Scanline Effect</span></label>
                                </div>
                            </div>
                        {/if}
                        {#if activeTab === "emulation"}
                            <!-- MODIFIED: Add ROMs Path setting -->
                            <div class="space-y-6">
                                <h3 class="text-lg font-semibold text-gray-300">Performance</h3>
                                <!-- CPU Clock Speed (unchanged) -->
                                <div>
                                    <label for="clockSpeed" class="block text-gray-400 text-sm font-medium mb-2">CPU Clock Speed: {$localSettings.clockSpeed} Hz</label>
                                    <input type="range" id="clockSpeed" min="100" max="2000" step="50" bind:value={$localSettings.clockSpeed} class="w-full h-2 bg-gray-700 rounded-lg appearance-none cursor-pointer" />
                                    <div class="flex flex-wrap gap-4 mt-2">
                                         <label class="inline-flex items-center"><input type="radio" class="form-radio" value={700} bind:group={$localSettings.clockSpeed} /><span class="ml-2">Original (700Hz)</span></label>
                                         <label class="inline-flex items-center"><input type="radio" class="form-radio" value={1400} bind:group={$localSettings.clockSpeed} /><span class="ml-2">Fast (1400Hz)</span></label>
                                         <label class="inline-flex items-center"><input type="radio" class="form-radio" value={2000} bind:group={$localSettings.clockSpeed} /><span class="ml-2">Turbo (2000Hz)</span></label>
                                    </div>
                                </div>
                                <div class="border-t border-gray-700 pt-4">
                                    <h3 class="text-lg font-semibold text-gray-300">Paths</h3>
                                     <!-- NEW: ROMs Path setting -->
                                    <div>
                                        <label for="romsPath" class="block text-gray-400 text-sm font-medium mb-2">ROMs Directory</label>
                                        <div class="flex gap-2">
                                            <input type="text" id="romsPath" bind:value={$localSettings.romsPath} class="w-full p-2 rounded-md bg-gray-700 border border-gray-600 text-gray-300 text-sm" readonly />
                                            <button on:click={browseForRomsPath} class="bg-blue-600 hover:bg-blue-700 text-white font-medium py-2 px-4 rounded-md transition-colors text-sm">Browse</button>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        {/if}
                         <!-- (Keybindings tab is unchanged) -->
                        {#if activeTab === "keybindings"}
                            <div>
                                <h3 class="text-lg font-semibold text-gray-300">Key Remapping</h3>
                                <p class="text-sm text-gray-400 mb-3">Click a key, then press the desired keyboard key to rebind.</p>
                                <div class="grid grid-cols-4 gap-3 text-center font-mono">
                                    {#each keybindings as binding (binding.chip8Key)}
                                        <div class="bg-gray-700 p-2 rounded-md border border-gray-600">
                                            <span class="font-bold text-gray-300">{binding.chip8Key.toString(16).toUpperCase()}</span>
                                            <input type="text" value={(binding.keyboardKey || "").toUpperCase()} on:focus={(e) => startRemap(e, binding.chip8Key)} on:blur={(e) => endRemap(e, binding.chip8Key)} class="w-full bg-gray-600 text-white text-center rounded-sm mt-1 focus:outline-none focus:ring-2 focus:ring-blue-500 p-1 cursor-pointer" readonly />
                                        </div>
                                    {/each}
                                </div>
                            </div>
                        {/if}
                    </div>
                </div>
            </div>
            <!-- ... (Save/Cancel buttons are unchanged) ... -->
            <div class="flex justify-end gap-3 mt-4 border-t border-gray-700 pt-4">
                <button on:click={closeModal} class="bg-gray-600 hover:bg-gray-500 text-white font-medium py-2 px-4 rounded-md transition-colors text-sm">Cancel</button>
                <button on:click={saveSettings} class="bg-blue-600 hover:bg-blue-500 text-white font-medium py-2 px-4 rounded-md transition-colors text-sm">Save & Close</button>
            </div>
        </div>
    </div>
{/if}
