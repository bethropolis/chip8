<script>
    import { onMount } from "svelte";
    import { EventsOn } from "./wailsjs/runtime/runtime.js";
    import {
        FrontendReady,
        GetInitialState,
        StartDebugUpdates, // Import new functions
        StopDebugUpdates,
    } from "./wailsjs/go/main/App.js";
    import { settings, initializeSettings } from "./lib/stores.js";
    import SettingsModal from "./lib/SettingsModal.svelte";
    import DebugPanel from "./lib/DebugPanel.svelte";
    import Notification from "./lib/Notification.svelte";
    import Header from "./lib/Header.svelte";
    import EmulatorView from "./lib/EmulatorView.svelte";

    /**
     * @typedef {Object} DebugState
     * @property {number[]} Registers
     * @property {any[]} Disassembly
     * @property {number[]} Stack
     * @property {Object} Breakpoints
     * @property {number} PC
     * @property {number} I
     * @property {number} SP
     * @property {number} DelayTimer
     * @property {number} SoundTimer
     */

    /** @type {DebugState} */
    let debugState = {
        Registers: Array(16).fill(0),
        Disassembly: [],
        Stack: Array(16).fill(0),
        Breakpoints: {},
        PC: 0,
        I: 0,
        SP: 0,
        DelayTimer: 0,
        SoundTimer: 0,
    };
    let statusMessage = "Status: Idle | ROM: None";
    let showSettingsModal = false;
    let currentTab = "emulator";

    // --- OPTIMIZATION: Reactive statement to control debug updates ---
    $: {
        if (currentTab === "debug") {
            StartDebugUpdates();
        } else {
            StopDebugUpdates();
        }
    }

    onMount(async () => {
        // We can stop debug updates on initial mount, just in case.
        StopDebugUpdates();

        EventsOn("debugUpdate", (newState) => {
            debugState = newState;
        });

        EventsOn("statusUpdate", (newStatus) => {
            statusMessage = newStatus;
        });

        await FrontendReady();

        const initialState = await GetInitialState();
        console.log("Initial state from backend:", initialState);
        if (initialState.cpuState) {
            debugState = initialState.cpuState;
        }
        if (initialState.settings) {
            initializeSettings(initialState.settings);
        } else {
            initializeSettings(null); // In case settings are not returned for some reason
        }
    });

    /** Open the settings modal. */
    function openSettings() {
        showSettingsModal = true;
    }

</script>
<div
    class="flex flex-col h-screen bg-gray-800 text-gray-200 font-sans antialiased"
>
    <Header bind:currentTab on:openSettings={openSettings} />

    <!-- Main Content Area -->
    <main class="flex-grow overflow-hidden">
        {#if currentTab === "emulator"}
            <EmulatorView />
        {:else if currentTab === "debug"}
            <DebugPanel bind:debugState />
        {/if}
    </main>
    <footer
        class="flex-none bg-gray-900 text-gray-400 text-xs text-center py-2 shadow-inner border-t border-gray-800"
    >
        {statusMessage}
    </footer>
    <Notification />
</div>
{#if showSettingsModal}
    <SettingsModal bind:showModal={showSettingsModal} />
{/if}
