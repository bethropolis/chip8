<script>
    import {
        WindowMinimise,
        WindowMaximise,
        WindowUnmaximise,
        WindowIsMaximised,
        Quit,
    } from "../wailsjs/runtime/runtime.js";
    import { Settings, Minimize, Maximize, X, Copy } from "lucide-svelte";
    import appIcon from "../assets/appicon.svg";
    import { createEventDispatcher, onMount } from "svelte";

    const dispatch = createEventDispatcher();

    export let currentTab;

    let isMaximized = false;

    onMount(async () => {
        isMaximized = await WindowIsMaximised();
    });

    async function toggleMaximize() {
        if (await WindowIsMaximised()) {
            WindowUnmaximise();
        } else {
            WindowMaximise();
        }
        isMaximized = !isMaximized;
    }

    function openSettings() {
        dispatch("openSettings");
    }
</script>

<header
    style="--wails-draggable:drag"
    class="flex-none bg-gray-900 text-gray-200 shadow-md z-20 flex items-center justify-between pr-2"
>
    <div class="flex items-center">
        <!-- App Icon and Title -->
        <div class="p-2 flex items-center space-x-2">
            <img src={appIcon} alt="App Icon" class="h-5 w-5" />
            <h1 class="text-md font-semibold text-gray-300">CHIP-8 Emulator</h1>
        </div>
        <!-- Tabs -->
        <nav class="flex space-x-1">
            <button
                on:click={() => (currentTab = "emulator")}
                class="px-3 py-1 rounded-md text-sm font-medium transition-colors duration-200"
                class:bg-gray-700={currentTab === "emulator"}
                class:text-white={currentTab === "emulator"}
                class:text-gray-400={currentTab !== "emulator"}
                class:hover:bg-gray-700={currentTab !== "emulator"}
                class:hover:text-white={currentTab !== "emulator"}
                >Emulator</button
            >
            <button
                on:click={() => (currentTab = "debug")}
                class="px-3 py-1 rounded-md text-sm font-medium transition-colors duration-200"
                class:bg-gray-700={currentTab === "debug"}
                class:text-white={currentTab === "debug"}
                class:text-gray-400={currentTab !== "debug"}
                class:hover:bg-gray-700={currentTab !== "debug"}
                class:hover:text-white={currentTab !== "debug"}
                >Debug</button
            >
        </nav>
    </div>

    <!-- Window Controls -->
    <div class="flex items-center space-x-1">
        <button
            on:click={openSettings}
            class="p-2 rounded-md hover:bg-gray-700 transition-colors duration-200"
            title="Settings"
        >
            <Settings size={16} />
        </button>
        <button
            on:click={WindowMinimise}
            class="p-2 rounded-md hover:bg-gray-700 transition-colors duration-200"
            title="Minimize"
        >
            <Minimize size={16} />
        </button>
        <button
            on:click={toggleMaximize}
            class="p-2 rounded-md hover:bg-gray-700 transition-colors duration-200"
            title={isMaximized ? "Restore" : "Maximize"}
        >
            {#if isMaximized}
                <Copy size={16} />
            {:else}
                <Maximize size={16} />
            {/if}
        </button>
        <button
            on:click={Quit}
            class="p-2 rounded-md hover:bg-red-600 transition-colors duration-200"
            title="Quit"
        >
            <X size={16} />
        </button>
    </div>
</header>
