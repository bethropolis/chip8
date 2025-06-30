<script>
    import { settings, updateAndSaveSettings } from "./stores.js";

    export let showModal;

    // --- Tab State ---
    let activeTab = "appearance";

    // --- FIX: Declare localSettings in the component's scope ---
    let localSettings = {};

    $: {
        if (showModal) {
            // Create a deep copy of the settings from the store
            // This prevents changes from affecting the global state until "Save" is clicked.
            localSettings = JSON.parse(JSON.stringify($settings));
        }
    }

    let remappingKey = null; // Stores the CHIP-8 key (a number, 0-15) being remapped

    // --- Derived State for Keybinding View ---
    let keybindings = [];
    $: {
        if (localSettings.keyMap) {
            // Create a display-friendly array from the localSettings
            keybindings = Array.from({ length: 16 }, (_, i) => {
                const chip8Hex = i;
                let keyboardKey = "N/A";
                for (const k in localSettings.keyMap) {
                    if (localSettings.keyMap[k] === chip8Hex) {
                        keyboardKey = k;
                        break;
                    }
                }
                return { chip8Key: chip8Hex, keyboardKey: keyboardKey };
            });
        }
    }

    function closeModal() {
        showModal = false;
    }

    async function saveSettings() {
        await updateAndSaveSettings(localSettings);
        closeModal();
    }

    // --- Key Remapping Logic ---
    function startRemap(event, chip8KeyToRemap) {
        // FIX: Assign the chip8 key correctly
        remappingKey = chip8KeyToRemap;
        event.target.value = "Press key...";
        window.addEventListener("keydown", handleRemapKeyDown, { once: true });
    }

    function handleRemapKeyDown(event) {
        event.preventDefault();
        if (remappingKey === null) return;

        const newKeyboardKey = event.key.toLowerCase();
        const updatedKeyMap = { ...localSettings.keyMap };
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

        localSettings = { ...localSettings, keyMap: updatedKeyMap };
        remappingKey = null;
    }

    function endRemap(event, chip8Key) {
        if (remappingKey !== null) {
            const originalKey = Object.keys($settings.keyMap).find(k => $settings.keyMap[k] === chip8Key) || 'N/A';
            event.target.value = originalKey.toUpperCase();
            remappingKey = null;
        }
    }
</script>

<!-- The HTML part of this component remains the same. -->
<!-- It will now correctly bind to the `localSettings` object. -->

{#if showModal}
    <div
        class="fixed inset-0 bg-black bg-opacity-70 flex items-center justify-center z-50 transition-opacity"
    >
        <div
            class="bg-gray-800 p-5 rounded-lg shadow-2xl border border-gray-700 w-full max-w-2xl"
        >
            <h2 class="text-xl font-semibold mb-4 text-center text-gray-200">
                Settings
            </h2>
            <div class="flex space-x-1">
                <!-- Sidebar -->
                <div class="w-1/4 bg-gray-900 p-3 rounded-l-md">
                    <nav class="space-y-1">
                        <button
                            on:click={() => (activeTab = "appearance")}
                            class="w-full text-left px-3 py-2 rounded-md text-sm font-medium transition-colors duration-150"
                            class:bg-gray-700={activeTab === "appearance"}
                            class:text-white={activeTab === "appearance"}
                            class:text-gray-400={activeTab !== "appearance"}
                            class:hover:bg-gray-700={activeTab !== "appearance"}
                            class:hover:text-white={activeTab !== "appearance"}
                            >Appearance</button
                        >
                        <button
                            on:click={() => (activeTab = "emulation")}
                            class="w-full text-left px-3 py-2 rounded-md text-sm font-medium transition-colors duration-150"
                            class:bg-gray-700={activeTab === "emulation"}
                            class:text-white={activeTab === "emulation"}
                            class:text-gray-400={activeTab !== "emulation"}
                            class:hover:bg-gray-700={activeTab !== "emulation"}
                            class:hover:text-white={activeTab !== "emulation"}
                            >Emulation</button
                        >
                        <button
                            on:click={() => (activeTab = "keybindings")}
                            class="w-full text-left px-3 py-2 rounded-md text-sm font-medium transition-colors duration-150"
                            class:bg-gray-700={activeTab === "keybindings"}
                            class:text-white={activeTab === "keybindings"}
                            class:text-gray-400={activeTab !== "keybindings"}
                            class:hover:bg-gray-700={activeTab !==
                                "keybindings"}
                            class:hover:text-white={activeTab !== "keybindings"}
                            >Keybindings</button
                        >
                    </nav>
                </div>
                <!-- Content -->
                <div class="w-3/4 bg-gray-800 p-4 rounded-r-md">
                    <div class="min-h-[300px]">
                        {#if activeTab === "appearance"}
                            <div class="space-y-5">
                                <h3 class="text-lg font-semibold text-gray-300">
                                    Display
                                </h3>
                                <div>
                                    <label
                                        class="block text-gray-400 text-sm font-medium mb-2"
                                        >Pixel Color</label
                                    >
                                    <div class="flex flex-wrap gap-4">
                                        <label class="inline-flex items-center"
                                            ><input
                                                type="radio"
                                                class="form-radio bg-gray-700 border-gray-600 text-green-500 focus:ring-green-500"
                                                name="displayColor"
                                                value="#33FF00"
                                                bind:group={
                                                    localSettings.displayColor
                                                }
                                            /><span class="ml-2"
                                                >Classic Green</span
                                            ></label
                                        >
                                        <label class="inline-flex items-center"
                                            ><input
                                                type="radio"
                                                class="form-radio bg-gray-700 border-gray-600 text-blue-500 focus:ring-blue-500"
                                                name="displayColor"
                                                value="#FFFFFF"
                                                bind:group={
                                                    localSettings.displayColor
                                                }
                                            /><span class="ml-2">White</span
                                            ></label
                                        >
                                        <label class="inline-flex items-center"
                                            ><input
                                                type="radio"
                                                class="form-radio bg-gray-700 border-gray-600 text-yellow-500 focus:ring-yellow-500"
                                                name="displayColor"
                                                value="#FFBF00"
                                                bind:group={
                                                    localSettings.displayColor
                                                }
                                            /><span class="ml-2">Amber</span
                                            ></label
                                        >
                                    </div>
                                </div>
                                <div>
                                    <label class="inline-flex items-center"
                                        ><input
                                            type="checkbox"
                                            class="form-checkbox bg-gray-700 border-gray-600 text-blue-500 focus:ring-blue-500"
                                            bind:checked={
                                                localSettings.scanlineEffect
                                            }
                                        /><span class="ml-2 text-gray-300"
                                            >Enable Scanline Effect</span
                                        ></label
                                    >
                                </div>
                            </div>
                        {/if}
                        {#if activeTab === "emulation"}
                            <div class="space-y-6">
                                <h3 class="text-lg font-semibold text-gray-300">
                                    Performance
                                </h3>
                                <div>
                                    <label
                                        for="clockSpeed"
                                        class="block text-gray-400 text-sm font-medium mb-2"
                                        >CPU Clock Speed: {localSettings.clockSpeed}
                                        Hz</label
                                    >
                                    <input
                                        type="range"
                                        id="clockSpeed"
                                        min="100"
                                        max="2000"
                                        step="50"
                                        bind:value={localSettings.clockSpeed}
                                        class="w-full h-2 bg-gray-700 rounded-lg appearance-none cursor-pointer"
                                    />
                                </div>
                                <div>
                                    <label
                                        class="block text-gray-400 text-sm font-medium mb-2"
                                        >Speed Presets</label
                                    >
                                    <div class="flex flex-wrap gap-4">
                                        <label class="inline-flex items-center"
                                            ><input
                                                type="radio"
                                                class="form-radio bg-gray-700 border-gray-600 text-blue-500 focus:ring-blue-500"
                                                value={700}
                                                bind:group={
                                                    localSettings.clockSpeed
                                                }
                                            /><span class="ml-2"
                                                >Original (700Hz)</span
                                            ></label
                                        >
                                        <label class="inline-flex items-center"
                                            ><input
                                                type="radio"
                                                class="form-radio bg-gray-700 border-gray-600 text-blue-500 focus:ring-blue-500"
                                                value={1400}
                                                bind:group={
                                                    localSettings.clockSpeed
                                                }
                                            /><span class="ml-2"
                                                >Fast (1400Hz)</span
                                            ></label
                                        >
                                        <label class="inline-flex items-center"
                                            ><input
                                                type="radio"
                                                class="form-radio bg-gray-700 border-gray-600 text-blue-500 focus:ring-blue-500"
                                                value={2000}
                                                bind:group={
                                                    localSettings.clockSpeed
                                                }
                                            /><span class="ml-2"
                                                >Turbo (2000Hz)</span
                                            ></label
                                        >
                                    </div>
                                </div>
                            </div>
                        {/if}
                        {#if activeTab === "keybindings"}
                            <div>
                                <h3 class="text-lg font-semibold text-gray-300">
                                    Key Remapping
                                </h3>
                                <p class="text-sm text-gray-400 mb-3">
                                    Click a key, then press the desired keyboard
                                    key to rebind.
                                </p>
                                <div
                                    class="grid grid-cols-4 gap-3 text-center font-mono"
                                >
                                    {#each keybindings as binding (binding.chip8Key)}
                                        <div
                                            class="bg-gray-700 p-2 rounded-md border border-gray-600"
                                        >
                                            <span
                                                class="font-bold text-gray-300"
                                                >{binding.chip8Key
                                                    .toString(16)
                                                    .toUpperCase()}</span
                                            >
                                            <input
                                                type="text"
                                                value={(
                                                    binding.keyboardKey || ""
                                                ).toUpperCase()}
                                                on:focus={(e) =>
                                                    startRemap(
                                                        e,
                                                        binding.chip8Key,
                                                    )}
                                                on:blur={(e) =>
                                                    endRemap(
                                                        e,
                                                        binding.chip8Key,
                                                    )}
                                                class="w-full bg-gray-600 text-white text-center rounded-sm mt-1 focus:outline-none focus:ring-2 focus:ring-blue-500 p-1 cursor-pointer"
                                                readonly
                                            />
                                        </div>
                                    {/each}
                                </div>
                            </div>
                        {/if}
                    </div>
                </div>
            </div>
            <!-- Action Buttons -->
            <div
                class="flex justify-end gap-3 mt-4 border-t border-gray-700 pt-4"
            >
                <button
                    on:click={closeModal}
                    class="bg-gray-600 hover:bg-gray-500 text-white font-medium py-2 px-4 rounded-md transition-colors text-sm"
                    >Cancel</button
                >
                <button
                    on:click={saveSettings}
                    class="bg-blue-600 hover:bg-blue-500 text-white font-medium py-2 px-4 rounded-md transition-colors text-sm"
                    >Save & Close</button
                >
            </div>
        </div>
    </div>
{/if}
