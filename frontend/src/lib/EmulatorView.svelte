<script>
    import {
        Camera, Pause, Play, RotateCcw, Save, Upload,
    } from "lucide-svelte";
    import { createEventDispatcher, onDestroy, onMount } from "svelte";
    import { settings, showNotification } from "./stores.js";
    import Gamepad from "svelte-gamepad";
    import {
        HardReset, KeyDown, KeyUp, LoadROM, LoadStateFromFile, SaveScreenshot, SaveStateToFile, SoftReset, TogglePause,
    } from "../wailsjs/go/main/App.js";
    import { EventsOn } from "../wailsjs/runtime/runtime.js";
    import { clickOutside } from "./clickOutside.js";
    import ROMBrowser from "./ROMBrowser.svelte";

    const dispatch = createEventDispatcher();
    $: keyMap = $settings.keyMap;
    $: currentDisplayColor = $settings.displayColor;
    $: currentScanlineEffect = $settings.scanlineEffect;

    let canvasElement;
    let isPaused = true;
    let currentDisplayBuffer = new Uint8Array(64 * 32);
    let showResetOptions = false;

    const keypadLayout = [
        { hex: 0x1, key: "1", keyboardKey: "1" }, { hex: 0x2, key: "2", keyboardKey: "2" }, { hex: 0x3, key: "3", keyboardKey: "3" }, { hex: 0xc, key: "C", keyboardKey: "4" },
        { hex: 0x4, key: "4", keyboardKey: "Q" }, { hex: 0x5, key: "5", keyboardKey: "W" }, { hex: 0x6, key: "6", keyboardKey: "E" }, { hex: 0xd, key: "D", keyboardKey: "R" },
        { hex: 0x7, key: "7", keyboardKey: "A" }, { hex: 0x8, key: "8", keyboardKey: "S" }, { hex: 0x9, key: "9", keyboardKey: "D" }, { hex: 0xe, key: "E", keyboardKey: "F" },
        { hex: 0xa, key: "A", keyboardKey: "Z" }, { hex: 0x0, key: "0", keyboardKey: "X" }, { hex: 0xb, key: "B", keyboardKey: "C" }, { hex: 0xf, key: "F", keyboardKey: "V" },
    ];
    const gamepadMap = { A: 0x5, B: 0x6, X: 0x8, Y: 0x9, DpadUp: 0x2, DpadDown: 0x8, DpadLeft: 0x7, DpadRight: 0x9, };
    let pressedKeys = {};

    $: scale = $settings.pixelScale || 10;

    const DISPLAY_WIDTH = 64;
    const DISPLAY_HEIGHT = 32;

    let audioContext;
    let oscillator;
    let animationFrameId;

    /**
     * Play a short beep sound using the Web Audio API.
     */
    function playBeep() {
        if (!audioContext) { audioContext = new (window.AudioContext || window.webkitAudioContext)(); }
        if (oscillator) { oscillator.stop(); oscillator.disconnect(); }
        oscillator = audioContext.createOscillator();
        oscillator.type = "sine";
        oscillator.frequency.setValueAtTime(440, audioContext.currentTime);
        oscillator.connect(audioContext.destination);
        oscillator.start();
        oscillator.stop(audioContext.currentTime + 0.1);
    }

    /**
     * Draw the CHIP-8 display buffer to the canvas.
     * @param {HTMLCanvasElement} canvas - The canvas element to draw on.
     * @param {Uint8Array} displayBuffer - The CHIP-8 display buffer (64x32).
     */
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
                    ctx.fillRect(x * scale, y * scale, scale, scale);
                }
            }
        }

        if (currentScanlineEffect) {
            ctx.fillStyle = "rgba(0, 0, 0, 0.3)";
            for (let y = 0; y < DISPLAY_HEIGHT; y += 2) {
                ctx.fillRect(0, y * scale, canvas.width, scale);
            }
        }
    }

    $: if (canvasElement) {
        drawDisplay(canvasElement, currentDisplayBuffer);
    }

    /**
     * Set up event listeners and initialize display on mount.
     */
    onMount(async () => {
        EventsOn("wails:file-drop", handleFileDrop);
        EventsOn("menu:pause", handleTogglePause);
        EventsOn("menu:savestate", handleSaveState);
        EventsOn("menu:softreset", handleSoftReset);
        EventsOn("menu:hardreset", handleHardReset);
        EventsOn("menu:loadstate", handleLoadState);
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
        EventsOn("playBeep", playBeep);
        drawDisplay(canvasElement, new Uint8Array(DISPLAY_WIDTH * DISPLAY_HEIGHT));
    });

    let reverseKeyMap = {};
    $: {
        reverseKeyMap = {};
        for (const [keyboardKey, chip8Key] of Object.entries($settings.keyMap)) {
            reverseKeyMap[keyboardKey] = chip8Key;
        }
    }

    /**
     * Listen for keyboard keydown events and map to CHIP-8 keys.
     */
    window.addEventListener("keydown", (e) => {
        const key = e.key.toLowerCase();
        const chip8Key = reverseKeyMap[key];
        if (chip8Key !== undefined) {
            e.preventDefault();
            KeyDown(chip8Key);
            pressedKeys = { ...pressedKeys, [chip8Key]: true };
        }
    });

    /**
     * Listen for keyboard keyup events and map to CHIP-8 keys.
     */
    window.addEventListener("keyup", (e) => {
        const key = e.key.toLowerCase();
        const chip8Key = reverseKeyMap[key];
        if (chip8Key !== undefined) {
            e.preventDefault();
            KeyUp(chip8Key);
            pressedKeys = { ...pressedKeys, [chip8Key]: false };
        }
    });

    /**
     * Toggle the paused state of the emulator.
     */
    async function handleTogglePause() { isPaused = await TogglePause(); }

    /**
     * Take a screenshot of the current canvas and save it.
     */
    async function handleScreenshot() {
        if (!canvasElement) { showNotification("Canvas not available for screenshot.", "error"); return; }
        try {
            const dataURL = canvasElement.toDataURL("image/png");
            const base64Data = dataURL.split(",")[1];
            await SaveScreenshot(base64Data);
            showNotification("Screenshot saved!", "success");
        } catch (error) { showNotification(`Failed to save screenshot: ${error}`, "error"); }
    }

    /**
     * Save the current emulator state to a file.
     */
    async function handleSaveState() {
        try {
            await SaveStateToFile();
            showNotification("Emulator state saved!", "success");
        } catch (error) { showNotification(`Failed to save state: ${error}`, "error"); }
    }

    /**
     * Load the emulator state from a file.
     */
    async function handleLoadState() {
        try {
            await LoadStateFromFile();
            showNotification("Emulator state loaded!", "success");
        } catch (error) { showNotification(`Failed to load state: ${error}`, "error"); }
    }

    /**
     * Toggle the visibility of reset options.
     */
    function toggleResetOptions() { showResetOptions = !showResetOptions; }

    /**
     * Perform a soft reset (reload ROM).
     */
    async function handleSoftReset() {
        try {
            await SoftReset();
            isPaused = false;
            showNotification("Soft reset complete! ROM reloaded.", "success");
        } catch (error) { showNotification(`Soft reset failed: ${error}`, "error"); }
        showResetOptions = false;
    }

    /**
     * Perform a hard reset (clear ROM).
     */
    async function handleHardReset() {
        try {
            await HardReset();
            isPaused = true;
            showNotification("Hard reset complete! ROM cleared.", "info");
        } catch (error) { showNotification(`Hard reset failed: ${error}`, "error"); }
        showResetOptions = false;
    }

    /**
     * Handle file drop event to load a ROM.
     * @param {Object} event - The file drop event.
     */
    async function handleFileDrop(event) {
        if (event.data.length > 0) {
            const romName = event.data[0].split("/").pop();
            try {
                await LoadROM(romName);
                showNotification(`Successfully loaded ${romName}`, "success");
            } catch (error) { showNotification(`Failed to load ROM: ${error}`, "error"); }
        }
    }

    /**
     * Show notification when a gamepad is connected.
     * @param {CustomEvent} e
     */
    function onGamepadConnected(e) { showNotification(`Gamepad ${e.detail.gamepadIndex + 1} connected.`, "success"); }

    /**
     * Show notification when a gamepad is disconnected.
     * @param {CustomEvent} e
     */
    function onGamepadDisconnected(e) { showNotification(`Gamepad ${e.detail.gamepadIndex + 1} disconnected.`, "warning"); }

    /**
     * Handle gamepad button events and map to CHIP-8 keys.
     * @param {CustomEvent} e
     */
    function handleGamepadButton(e) {
        const chip8Key = gamepadMap[e.type];
        if (chip8Key !== undefined) {
            if (e.detail.pressed) { handleKeypadPress(chip8Key); } else { handleKeypadRelease(chip8Key); }
        }
    }

    /**
     * Handle keypad press for a given CHIP-8 key.
     * @param {number} key
     */
    function handleKeypadPress(key) {
        KeyDown(key);
        pressedKeys = { ...pressedKeys, [key]: true };
    }

    /**
     * Handle keypad release for a given CHIP-8 key.
     * @param {number} key
     */
    function handleKeypadRelease(key) {
        KeyUp(key);
        pressedKeys = { ...pressedKeys, [key]: false };
    }
</script>

<Gamepad on:Connected={onGamepadConnected} on:Disconnected={onGamepadDisconnected} on:A={handleGamepadButton} on:B={handleGamepadButton} on:X={handleGamepadButton} on:Y={handleGamepadButton} on:DpadUp={handleGamepadButton} on:DpadDown={handleGamepadButton} on:DpadLeft={handleGamepadButton} on:DpadRight={handleGamepadButton} />

<div class="flex flex-col md:flex-row h-full p-3 space-y-3 md:space-y-0 md:space-x-3">
    <section class="flex-grow flex items-center justify-center bg-gray-900 rounded-md shadow-inner p-3">
        <canvas
            bind:this={canvasElement}
            width={DISPLAY_WIDTH * scale}
            height={DISPLAY_HEIGHT * scale}
            class="border border-gray-700 rounded-sm"
        ></canvas>
    </section>
    <aside class="flex-none w-full md:w-72 flex flex-col space-y-3">
        <ROMBrowser />
        <div class="bg-gray-900 p-3 rounded-md shadow-inner">
            <h2 class="text-lg font-semibold mb-2 text-center text-gray-400">Controls</h2>
            <div class="grid grid-cols-2 gap-2">
                <div class="relative inline-block text-left" use:clickOutside={() => (showResetOptions = false)}>
                    <button on:click={toggleResetOptions} class="flex items-center justify-center space-x-2 bg-yellow-600 hover:bg-yellow-700 text-white font-medium py-2 px-3 rounded-md transition-colors duration-200 w-full text-sm" title="Reset Options">
                        <RotateCcw size={16} />
                        <span>Reset</span>
                    </button>
                    {#if showResetOptions}
                        <div class="origin-top-right absolute right-0 mt-1 w-full rounded-md shadow-lg bg-gray-700 ring-1 ring-black ring-opacity-5 focus:outline-none z-10">
                            <div class="py-1">
                                <button on:click={handleSoftReset} class="block w-full text-left px-3 py-1 text-sm text-gray-200 hover:bg-gray-600">Soft Reset</button>
                                <button on:click={handleHardReset} class="block w-full text-left px-3 py-1 text-sm text-gray-200 hover:bg-gray-600">Hard Reset</button>
                            </div>
                        </div>
                    {/if}
                </div>
                <button on:click={handleTogglePause} class="flex items-center justify-center space-x-2 bg-green-600 hover:bg-green-700 text-white font-medium py-2 px-3 rounded-md transition-colors duration-200 text-sm" title={isPaused ? "Resume emulation (Ctrl+P)" : "Pause emulation (Ctrl+P)"}>
                    {#if isPaused}<Play size={16} /><span>Resume</span>{:else}<Pause size={16} /><span>Pause</span>{/if}
                </button>
                <button on:click={handleScreenshot} class="flex items-center justify-center space-x-2 bg-blue-600 hover:bg-blue-700 text-white font-medium py-2 px-3 rounded-md transition-colors duration-200 col-span-2 text-sm" title="Take a screenshot"><Camera size={16} /><span>Screenshot</span></button>
                <button on:click={handleSaveState} class="flex items-center justify-center space-x-2 bg-indigo-600 hover:bg-indigo-700 text-white font-medium py-2 px-3 rounded-md transition-colors duration-200 text-sm" title="Save State"><Save size={16} /><span>Save State</span></button>
                <button on:click={handleLoadState} class="flex items-center justify-center space-x-2 bg-indigo-600 hover:bg-indigo-700 text-white font-medium py-2 px-3 rounded-md transition-colors duration-200 text-sm" title="Load State (Ctrl+O)"><Upload size={16} /><span>Load State</span></button>
            </div>
        </div>
        <div class="bg-gray-900 p-3 rounded-md shadow-inner">
            <h2 class="text-lg font-semibold mb-2 text-center text-gray-400">CHIP-8 Keypad</h2>
            <div class="grid grid-cols-4 gap-2 text-center font-mono">
                {#each keypadLayout as { hex, key, keyboardKey } (hex)}
                    <button on:mousedown={() => handleKeypadPress(hex)} on:mouseup={() => handleKeypadRelease(hex)} on:mouseleave={() => handleKeypadRelease(hex)} on:touchstart|preventDefault={() => handleKeypadPress(hex)} on:touchend|preventDefault={() => handleKeypadRelease(hex)} on:touchcancel|preventDefault={() => handleKeypadRelease(hex)} class="p-2 rounded-md border text-lg font-bold flex flex-col items-center justify-center aspect-square transition-all duration-100 focus:outline-none" class:bg-blue-500={pressedKeys[hex]} class:border-blue-400={pressedKeys[hex]} class:text-white={pressedKeys[hex]} class:bg-gray-700={!pressedKeys[hex]} class:border-gray-600={!pressedKeys[hex]} class:hover:bg-gray-600={!pressedKeys[hex]} title={`CHIP-8 Key: ${hex.toString(16).toUpperCase()}`}>
                        <span class="text-xl">{key}</span>
                        <span class="text-xs text-gray-400 mt-1">{keyboardKey}</span>
                    </button>
                {/each}
            </div>
        </div>
    </aside>
</div>
