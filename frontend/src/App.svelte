<script>
    import { onMount } from 'svelte';
    import { EventsOn, OnFileDrop } from './wailsjs/runtime/runtime.js';
    import {
        FrontendReady,
        GetInitialState,
        StartDebugUpdates,
        StopDebugUpdates,
        LoadROMByPath,
        TogglePause
    } from "./wailsjs/go/main/App.js";
    import { settings, initializeSettings, showNotification } from "./lib/stores.js";
    import SettingsModal from "./lib/SettingsModal.svelte";
    import DebugPanel from "./lib/DebugPanel.svelte";
    import Notification from "./lib/Notification.svelte";
    import Header from "./lib/Header.svelte";
    import EmulatorView from "./lib/EmulatorView.svelte";

    let debugState = {};
    let showSettingsModal = false;
    let currentTab = "emulator";

    let isPaused = true;
    let romName = "None";
    let clockSpeed = 700;
    $: statusMessage = `Status: ${isPaused ? 'Paused' : 'Running'} | ROM: ${romName} | Speed: ${clockSpeed} Hz`;

    $: {
        if (currentTab === "debug") {
            StartDebugUpdates();
        } else {
            StopDebugUpdates();
        }
    }

    onMount(async () => {
        StopDebugUpdates();

        EventsOn("debugUpdate", (newState) => {
            debugState = newState;
        });

        EventsOn("statusUpdate", (newStatus) => {
            const parts = newStatus.split("|");
            if (parts.length > 1 && parts[1].includes("ROM:")) {
                romName = parts[1].replace("ROM:", "").trim();
            }
        });

        EventsOn("pauseUpdate", (pausedState) => {
            isPaused = pausedState;
        });

        EventsOn("menu:pause", async () => {
           isPaused = await TogglePause();
        });

        EventsOn("clockSpeedUpdate", (speed) => {
            clockSpeed = speed;
        });

        OnFileDrop((x, y, paths) => {
            if (paths.length > 0) {
                const fullPath = paths[0];
                LoadROMByPath(fullPath).then((loadedRomName) => {
                    romName = loadedRomName;
                    isPaused = false;
                    showNotification(`Loaded ${romName} via drop!`, 'success');
                }).catch(err => {
                    showNotification(`Failed to load dropped ROM: ${err}`, 'error');
                });
            }
        }, false);

        await FrontendReady();
        const initialState = await GetInitialState();
        if (initialState.cpuState) {
            debugState = initialState.cpuState;
        }
        if (initialState.settings) {
            initializeSettings(initialState.settings);
            clockSpeed = initialState.settings.clockSpeed || 700;
        } else {
            initializeSettings(null);
        }
    });

    function openSettings() {
        showSettingsModal = true;
    }
</script>

<div
    class="flex flex-col h-screen bg-gray-800 text-gray-200 font-sans antialiased"
    style="--wails-drop-target: drag;"
>
    <Header bind:currentTab on:openSettings={openSettings} />

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
