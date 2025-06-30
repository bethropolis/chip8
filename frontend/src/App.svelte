<script>
    import { onMount, onDestroy } from "svelte";
    import {
        EventsOn,
        WindowFullscreen,
        WindowUnfullscreen,
        WindowIsFullscreen,
        WindowMinimise,
        WindowUnminimise,
        WindowMaximise,
        WindowUnmaximise,
        WindowIsMaximised,
        Quit,
    } from "./wailsjs/runtime/runtime.js";
    import {
        TogglePause,
        KeyDown,
        KeyUp,
        FrontendReady,
        GetInitialState,
        SetClockSpeed,
        SaveScreenshot,
        SaveState,
        SaveStateToFile,
        LoadStateFromFile,
        SoftReset,
        HardReset,
    } from "./wailsjs/go/main/App.js";
    import SettingsModal from "./lib/SettingsModal.svelte";
    import DebugPanel from "./lib/DebugPanel.svelte";
    import Notification from "./lib/Notification.svelte";
    import ROMBrowser from "./lib/ROMBrowser.svelte";
    import {
        Settings,
        RotateCcw,
        Play,
        Pause,
        Camera,
        Save,
        Upload,
        Minimize,
        Maximize,
        X,
        Copy,
    } from "lucide-svelte";

    // --- UI Elements & State ---
    let canvasElement;
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
    let isPaused = true;
    let showSettingsModal = false;
    let currentClockSpeed = 700;
    let currentDisplayColor = "#33FF00";
    let currentScanlineEffect = false;
    let currentDisplayScale = 1;
    let currentTab = "emulator";
    let currentDisplayBuffer = new Uint8Array(64 * 32);

    let notificationMessage = "";
    let notificationType = "info";
    let showResetOptions = false;
    let isMaximized = false;

    const keypadLayout = [
        { hex: 0x1, key: "1", keyboardKey: "1" },
        { hex: 0x2, key: "2", keyboardKey: "2" },
        { hex: 0x3, key: "3", keyboardKey: "3" },
        { hex: 0xc, key: "C", keyboardKey: "4" },
        { hex: 0x4, key: "4", keyboardKey: "Q" },
        { hex: 0x5, key: "5", keyboardKey: "W" },
        { hex: 0x6, key: "6", keyboardKey: "E" },
        { hex: 0xd, key: "D", keyboardKey: "R" },
        { hex: 0x7, key: "7", keyboardKey: "A" },
        { hex: 0x8, key: "8", keyboardKey: "S" },
        { hex: 0x9, key: "9", keyboardKey: "D" },
        { hex: 0xe, key: "E", keyboardKey: "F" },
        { hex: 0xa, key: "A", keyboardKey: "Z" },
        { hex: 0x0, key: "0", keyboardKey: "X" },
        { hex: 0xb, key: "B", keyboardKey: "C" },
        { hex: 0xf, key: "F", keyboardKey: "V" },
    ];

    // --- Display Constants ---
    const SCALE = 10;
    const DISPLAY_WIDTH = 64;
    const DISPLAY_HEIGHT = 32;

    // --- Keypad Mapping ---
    let keyMap = {
        "1": 0x1,
        "2": 0x2,
        "3": 0x3,
        "4": 0xc,
        q: 0x4,
        w: 0x5,
        e: 0x6,
        r: 0xd,
        a: 0x7,
        s: 0x8,
        d: 0x9,
        f: 0xe,
        z: 0xa,
        x: 0x0,
        c: 0xb,
        v: 0xf,
    };
    let pressedKeys = {};

    // --- Audio ---
    let audioContext;
    let oscillator;
    let animationFrameId;

    function playBeep() {
        if (!audioContext) {
            audioContext = new (window.AudioContext ||
                window.webkitAudioContext)();
        }
        if (oscillator) {
            oscillator.stop();
            oscillator.disconnect();
        }
        oscillator = audioContext.createOscillator();
        oscillator.type = "sine";
        oscillator.frequency.setValueAtTime(440, audioContext.currentTime);
        oscillator.connect(audioContext.destination);
        oscillator.start();
        oscillator.stop(audioContext.currentTime + 0.1);
    }

    function drawDisplay(canvas, displayBuffer) {
        if (!canvas || !displayBuffer) return;
        const ctx = canvas.getContext("2d");
        if (!ctx) return;

        ctx.fillStyle = "#000000";
        ctx.fillRect(0, 0, canvas.width, canvas.height);

        ctx.fillStyle = currentDisplayColor;
        for (let y = 0; y < DISPLAY_HEIGHT; y++) {
            for (let x = 0; x < DISPLAY_WIDTH; x++) {
                if (displayBuffer[y * DISPLAY_WIDTH + x]) {
                    ctx.fillRect(x * SCALE, y * SCALE, SCALE, SCALE);
                }
            }
        }

        if (currentScanlineEffect) {
            ctx.fillStyle = "rgba(0, 0, 0, 0.3)";
            for (let y = 0; y < DISPLAY_HEIGHT; y += 2) {
                ctx.fillRect(0, y * SCALE, canvas.width, SCALE);
            }
        }
    }

    onMount(async () => {
        isMaximized = await WindowIsMaximised();

        EventsOn("wails:file-drop", handleFileDrop);
        EventsOn("menu:loadrom", handleLoadState);
        EventsOn("menu:pause", handleTogglePause);

        EventsOn("displayUpdate", (base64DisplayBuffer) => {
            if (animationFrameId) cancelAnimationFrame(animationFrameId);
            animationFrameId = requestAnimationFrame(() => {
                const binaryString = atob(base64DisplayBuffer);
                const bytes = new Uint8Array(binaryString.length);
                for (let i = 0; i < binaryString.length; i++) {
                    bytes[i] = binaryString.charCodeAt(i);
                }
                currentDisplayBuffer = bytes;
                drawDisplay(canvasElement, currentDisplayBuffer);
            });
        });

        EventsOn("debugUpdate", (newState) => {
            debugState = newState;
        });

        EventsOn("statusUpdate", (newStatus) => {
            statusMessage = newStatus;
        });

        EventsOn("clockSpeedUpdate", (speed) => {
            currentClockSpeed = speed;
        });

        EventsOn("playBeep", playBeep);

        await FrontendReady();

        const initialState = await GetInitialState();
        debugState = initialState;
        drawDisplay(
            canvasElement,
            new Uint8Array(DISPLAY_WIDTH * DISPLAY_HEIGHT)
        );
    });

    // Redraw when canvas or tab changes
    $: if (canvasElement && currentTab === "emulator") {
        drawDisplay(canvasElement, currentDisplayBuffer);
    }

    // Reverse map for finding CHIP-8 key from keyboard key
    let reverseKeyMap = {};
    $: {
        reverseKeyMap = {};
        for (const [keyboardKey, chip8Key] of Object.entries(keyMap)) {
            reverseKeyMap[keyboardKey] = chip8Key;
        }
    }

    window.addEventListener("keydown", (e) => {
        const key = e.key.toLowerCase();
        const chip8Key = reverseKeyMap[key];
        if (chip8Key !== undefined) {
            e.preventDefault();
            KeyDown(chip8Key);
            pressedKeys = { ...pressedKeys, [chip8Key]: true };
        }
    });

    window.addEventListener("keyup", (e) => {
        const key = e.key.toLowerCase();
        const chip8Key = reverseKeyMap[key];
        if (chip8Key !== undefined) {
            e.preventDefault();
            KeyUp(chip8Key);
            pressedKeys = { ...pressedKeys, [chip8Key]: false };
        }
    });

    async function handleTogglePause() {
        isPaused = await TogglePause();
    }

    function openSettings() {
        showSettingsModal = true;
    }

    async function handleSaveSettings(event) {
        const {
            clockSpeed,
            displayColor,
            scanlineEffect,
            keyMap: newKeyMap,
        } = event.detail;
        await SetClockSpeed(clockSpeed);
        currentClockSpeed = clockSpeed;
        currentDisplayColor = displayColor;
        currentScanlineEffect = scanlineEffect;
        keyMap = newKeyMap;
    }

    async function handleScreenshot() {
        if (!canvasElement) {
            showNotification("Canvas not available for screenshot.", "error");
            return;
        }
        try {
            const dataURL = canvasElement.toDataURL("image/png");
            const base64Data = dataURL.split(",")[1];
            await SaveScreenshot(base64Data);
            showNotification("Screenshot saved!", "success");
        } catch (error) {
            showNotification(`Failed to save screenshot: ${error}`, "error");
        }
    }

    async function handleSaveState() {
        try {
            const state = await SaveState();
            await SaveStateToFile(state);
            showNotification("Emulator state saved!", "success");
        } catch (error) {
            showNotification(`Failed to save state: ${error}`, "error");
        }
    }

    async function handleLoadState() {
        try {
            await LoadStateFromFile();
            showNotification("Emulator state loaded!", "success");
        } catch (error) {
            showNotification(`Failed to load state: ${error}`, "error");
        }
    }

    export function showNotification(message, type = "info") {
        notificationMessage = message;
        notificationType = type;
    }

    function dismissNotification() {
        notificationMessage = "";
    }

    function toggleResetOptions() {
        showResetOptions = !showResetOptions;
    }

    async function handleSoftReset() {
        try {
            await SoftReset();
            isPaused = false;
            showNotification("Soft reset complete! ROM reloaded.", "success");
        } catch (error) {
            showNotification(`Soft reset failed: ${error}`, "error");
        }
        showResetOptions = false;
    }

    async function handleHardReset() {
        try {
            await HardReset();
            isPaused = true;
            showNotification("Hard reset complete! ROM cleared.", "info");
        } catch (error) {
            showNotification(`Hard reset failed: ${error}`, "error");
        }
        showResetOptions = false;
    }

    async function handleFileDrop(event) {
        if (event.data.length > 0) {
            const romName = event.data[0].split("/").pop();
            try {
                await LoadROM(romName);
                showNotification(`Successfully loaded ${romName}`, "success");
            } catch (error) {
                showNotification(`Failed to load ROM: ${error}`, "error");
            }
        }
    }

    async function toggleMaximize() {
        if (await WindowIsMaximised()) {
            WindowUnmaximise();
        } else {
            WindowMaximise();
        }
        isMaximized = !isMaximized;
    }

    function handleKeypadPress(key) {
        KeyDown(key);
        pressedKeys = { ...pressedKeys, [key]: true };
    }

    function handleKeypadRelease(key) {
        KeyUp(key);
        pressedKeys = { ...pressedKeys, [key]: false };
    }
</script>

    <div
        class="flex flex-col h-screen bg-gray-800 text-gray-200 font-sans antialiased"
    >
        <!-- Custom Title Bar -->
        <header
            style="--wails-draggable:drag"
            class="flex-none bg-gray-900 text-gray-200 shadow-md z-20 flex items-center justify-between pr-2"
        >
            <div class="flex items-center">
                <!-- App Icon and Title -->
                <div class="p-2 flex items-center space-x-2">
                    <img src="/appicon.png" alt="App Icon" class="h-5 w-5" />
                    <h1 class="text-md font-semibold text-gray-300">
                        CHIP-8 Emulator
                    </h1>
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

        <!-- Main Content Area -->
        <main class="flex-grow overflow-hidden">
            {#if currentTab === "emulator"}
                <div
                    class="flex flex-col md:flex-row h-full p-3 space-y-3 md:space-y-0 md:space-x-3"
                >
                    <section
                        class="flex-grow flex items-center justify-center bg-gray-900 rounded-md shadow-inner p-3"
                    >
                        <canvas
                            bind:this={canvasElement}
                            width={DISPLAY_WIDTH * SCALE}
                            height={DISPLAY_HEIGHT * SCALE}
                            class="border border-gray-700 rounded-sm"
                        ></canvas>
                    </section>
                    <aside
                        class="flex-none w-full md:w-72 flex flex-col space-y-3"
                    >
                        <ROMBrowser {showNotification} />
                        <div class="bg-gray-900 p-3 rounded-md shadow-inner">
                            <h2
                                class="text-lg font-semibold mb-2 text-center text-gray-400"
                            >
                                Controls
                            </h2>
                            <div class="grid grid-cols-2 gap-2">
                                <div class="relative inline-block text-left">
                                    <button
                                        on:click={toggleResetOptions}
                                        class="flex items-center justify-center space-x-2 bg-yellow-600 hover:bg-yellow-700 text-white font-medium py-2 px-3 rounded-md transition-colors duration-200 w-full text-sm"
                                        title="Reset Options"
                                    >
                                        <RotateCcw size={16} />
                                        <span>Reset</span>
                                    </button>
                                    {#if showResetOptions}
                                        <div
                                            class="origin-top-right absolute right-0 mt-1 w-full rounded-md shadow-lg bg-gray-700 ring-1 ring-black ring-opacity-5 focus:outline-none z-10"
                                        >
                                            <div class="py-1">
                                                <button
                                                    on:click={handleSoftReset}
                                                    class="block w-full text-left px-3 py-1 text-sm text-gray-200 hover:bg-gray-600"
                                                    >Soft Reset</button
                                                >
                                                <button
                                                    on:click={handleHardReset}
                                                    class="block w-full text-left px-3 py-1 text-sm text-gray-200 hover:bg-gray-600"
                                                    >Hard Reset</button
                                                >
                                            </div>
                                        </div>
                                    {/if}
                                </div>
                                <button
                                    on:click={handleTogglePause}
                                    class="flex items-center justify-center space-x-2 bg-green-600 hover:bg-green-700 text-white font-medium py-2 px-3 rounded-md transition-colors duration-200 text-sm"
                                    title={isPaused
                                        ? "Resume emulation (Ctrl+P)"
                                        : "Pause emulation (Ctrl+P)"}
                                >
                                    {#if isPaused}<Play size={16} /><span
                                            >Resume</span
                                        >{:else}<Pause size={16} /><span
                                            >Pause</span
                                        >{/if}
                                </button>
                                <button
                                    on:click={handleScreenshot}
                                    class="flex items-center justify-center space-x-2 bg-blue-600 hover:bg-blue-700 text-white font-medium py-2 px-3 rounded-md transition-colors duration-200 col-span-2 text-sm"
                                    title="Take a screenshot"
                                    ><Camera size={16} /><span
                                        >Screenshot</span
                                    ></button
                                >
                                <button
                                    on:click={handleSaveState}
                                    class="flex items-center justify-center space-x-2 bg-indigo-600 hover:bg-indigo-700 text-white font-medium py-2 px-3 rounded-md transition-colors duration-200 text-sm"
                                    title="Save State"
                                    ><Save size={16} /><span>Save State</span
                                    ></button
                                >
                                <button
                                    on:click={handleLoadState}
                                    class="flex items-center justify-center space-x-2 bg-indigo-600 hover:bg-indigo-700 text-white font-medium py-2 px-3 rounded-md transition-colors duration-200 text-sm"
                                    title="Load State (Ctrl+O)"
                                    ><Upload size={16} /><span
                                        >Load State</span
                                    ></button
                                >
                            </div>
                        </div>
                        <div class="bg-gray-900 p-3 rounded-md shadow-inner">
                            <h2
                                class="text-lg font-semibold mb-2 text-center text-gray-400"
                            >
                                CHIP-8 Keypad
                            </h2>
                            <div
                                class="grid grid-cols-4 gap-2 text-center font-mono"
                            >
                                {#each keypadLayout as { hex, key, keyboardKey } (hex)}
                                    <button
                                        on:mousedown={() => handleKeypadPress(hex)}
                                        on:mouseup={() => handleKeypadRelease(hex)}
                                        on:mouseleave={() => handleKeypadRelease(hex)} 
                                        class="p-2 rounded-md border text-lg font-bold flex flex-col items-center justify-center aspect-square transition-all duration-100 focus:outline-none"
                                        class:bg-blue-500={pressedKeys[hex]}
                                        class:border-blue-400={pressedKeys[hex]}
                                        class:text-white={pressedKeys[hex]}
                                        class:bg-gray-700={!pressedKeys[hex]}
                                        class:border-gray-600={!pressedKeys[hex]}
                                        class:hover:bg-gray-600={!pressedKeys[hex]}
                                        title={`CHIP-8 Key: ${hex.toString(16).toUpperCase()}`}
                                    >
                                        <span class="text-xl">{key}</span>
                                        <span class="text-xs text-gray-400 mt-1"
                                            >{keyboardKey}</span
                                        >
                                    </button>
                                {/each}
                            </div>
                        </div>
                    </aside>
                </div>
            {:else if currentTab === "debug"}
                <DebugPanel bind:debugState />
            {/if}
        </main>
        <footer
            class="flex-none bg-gray-900 text-gray-400 text-xs text-center py-2 shadow-inner border-t border-gray-800"
        >
            {statusMessage}
        </footer>
        <Notification
            message={notificationMessage}
            type={notificationType}
            on:dismiss={dismissNotification}
        />
    </div>
{#if showSettingsModal}
    <SettingsModal
        bind:showModal={showSettingsModal}
        {currentClockSpeed}
        {currentDisplayColor}
        {currentScanlineEffect}
        {currentDisplayScale}
        currentKeyMap={keyMap}
        on:save={handleSaveSettings}
    />
{/if}
