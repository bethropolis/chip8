<script>
    import { onMount, onDestroy } from 'svelte';
    import { GetMemory, GetLogs, SetBreakpoint, ClearBreakpoint } from '../wailsjs/go/main/App';
    import { EventsOn } from '../wailsjs/runtime/runtime.js';
    import LogViewer from './LogViewer.svelte';

    export let debugState;

    let memoryData = new Uint8Array(256);
    let memoryOffset = 0x200; // Start at program memory
    let memoryLimit = 256;
    let memoryUpdateInterval;

    async function fetchMemoryView() {
        if (!debugState.PC) return; // Don't fetch if no ROM is loaded
        try {
            const data = await GetMemory(memoryOffset, memoryLimit);
            if (data) {
                memoryData = new Uint8Array(atob(data).split('').map(char => char.charCodeAt(0)));
            }
        } catch (error) {
            console.error("Failed to fetch memory view:", error);
        }
    }

    onMount(() => {
        memoryUpdateInterval = setInterval(fetchMemoryView, 200);
    });

    onDestroy(() => {
        clearInterval(memoryUpdateInterval);
    });

    function formatByte(byte) {
        return byte.toString(16).padStart(2, '0').toUpperCase();
    }

    function formatAddress(address) {
        return '0x' + address.toString(16).padStart(4, '0').toUpperCase();
    }

    function handleMemoryScroll(event) {
        const target = event.target;
        if (event.deltaY > 0) {
            memoryOffset += 16;
        } else {
            memoryOffset -= 16;
        }

        if (memoryOffset < 0) memoryOffset = 0;
        if (memoryOffset > 4096 - memoryLimit) memoryOffset = 4096 - memoryLimit;

        event.preventDefault();
        fetchMemoryView();
    }

    async function toggleBreakpoint(address) {
        if (debugState.Breakpoints && debugState.Breakpoints[address]) {
            await ClearBreakpoint(address);
        } else {
            await SetBreakpoint(address);
        }
    }
</script>

<div class="grid grid-cols-1 lg:grid-cols-3 gap-3 p-3 h-full overflow-y-auto bg-gray-900 text-gray-300 font-sans">
    
    <!-- Left Column -->
    <div class="lg:col-span-1 flex flex-col space-y-3">
        <!-- CPU State -->
        <div class="bg-gray-800 p-3 rounded-md border border-gray-700">
            <h3 class="font-semibold text-md mb-2 text-gray-400">CPU State</h3>
            <div class="grid grid-cols-2 gap-x-4 text-sm font-mono">
                <p>PC: <span class="text-cyan-400">{formatAddress(debugState.PC ?? 0)}</span></p>
                <p>I: <span class="text-cyan-400">{formatAddress(debugState.I ?? 0)}</span></p>
                <p>SP: <span class="text-cyan-400">{formatAddress(debugState.SP ?? 0)}</span></p>
            </div>
        </div>

        <!-- Timers -->
        <div class="bg-gray-800 p-3 rounded-md border border-gray-700">
            <h3 class="font-semibold text-md mb-2 text-gray-400">Timers</h3>
            <div class="grid grid-cols-2 gap-x-4 text-sm font-mono">
                <p>Delay: <span class="text-green-400">{debugState.DelayTimer ?? 0}</span></p>
                <p>Sound: <span class="text-green-400">{debugState.SoundTimer ?? 0}</span></p>
            </div>
        </div>

        <!-- Registers -->
        <div class="bg-gray-800 p-3 rounded-md border border-gray-700">
            <h3 class="font-semibold text-md mb-2 text-gray-400">Registers</h3>
            <div class="grid grid-cols-4 gap-x-2 gap-y-1 text-sm font-mono">
                {#each { length: 16 } as _, i}
                    <span>V{i.toString(16).toUpperCase()}: <span class="text-yellow-400">{`0x${debugState.Registers?.[i]?.toString(16).padStart(2, "0").toUpperCase() ?? "00"}`}</span></span>
                {/each}
            </div>
        </div>

        <!-- Stack -->
        <div class="bg-gray-800 p-3 rounded-md border border-gray-700">
            <h3 class="font-semibold text-md mb-2 text-gray-400">Stack</h3>
            <pre class="text-xs overflow-y-auto h-28 bg-gray-900 p-2 rounded-md border border-gray-700 font-mono">
                {#each debugState.Stack || [] as value, i}
                    <div class:text-cyan-300={i === (debugState.SP > 0 ? debugState.SP -1 : 0)} class:font-bold={i === (debugState.SP > 0 ? debugState.SP -1 : 0)}>Stack[{i.toString(16).toUpperCase()}]: {formatAddress(value)}</div>
                {/each}
            </pre>
        </div>
    </div>

    <!-- Middle Column -->
    <div class="lg:col-span-1 flex flex-col space-y-3">
        <!-- Disassembly -->
        <div class="bg-gray-800 p-3 rounded-md border border-gray-700 flex-grow flex flex-col">
            <h3 class="font-semibold text-md mb-2 text-gray-400">Disassembly</h3>
            <pre class="text-xs leading-snug overflow-y-auto bg-gray-900 p-2 rounded-md border border-gray-700 flex-grow font-mono">
                {#each debugState.Disassembly || [] as line}
                    {@const address = parseInt(line.split(":")[0].replace("► ", ""), 16)}
                    <div
                        class="cursor-pointer hover:bg-gray-700 px-1 rounded-sm"
                        class:text-cyan-300={line.startsWith("►")}
                        class:font-bold={line.startsWith("►")}
                        class:bg-red-800={debugState.Breakpoints && debugState.Breakpoints[address]}
                        class:hover:bg-red-700={debugState.Breakpoints && debugState.Breakpoints[address]}
                        on:click={() => toggleBreakpoint(address)}
                        title="Click to toggle breakpoint"
                    >{line}</div>
                {/each}
            </pre>
        </div>
    </div>

    <!-- Right Column -->
    <div class="lg:col-span-1 flex flex-col space-y-3">
        <!-- Memory Viewer -->
        <div class="bg-gray-800 p-3 rounded-md border border-gray-700 flex-grow flex flex-col">
            <h3 class="font-semibold text-md mb-2 text-gray-400">Memory Viewer</h3>
            <div class="text-xs overflow-y-auto bg-gray-900 p-2 rounded-md border border-gray-700 flex-grow font-mono" on:wheel={handleMemoryScroll}>
                {#each Array(Math.ceil(memoryData.length / 16)) as _, rowIdx}
                    <div class="flex whitespace-pre">
                        <span class="text-gray-500 mr-2">{formatAddress(memoryOffset + rowIdx * 16)}:</span>
                        <div class="flex-grow grid grid-cols-16">
                            {#each Array(16) as _, colIdx}
                                {@const byte = memoryData[rowIdx * 16 + colIdx]}
                                <span class="mr-1">{byte !== undefined ? formatByte(byte) : "--"}</span>
                            {/each}
                        </div>
                    </div>
                {/each}
            </div>
        </div>
    </div>

    <!-- Logs (Full Width) -->
    <div class="lg:col-span-3 bg-gray-800 p-3 rounded-md border border-gray-700">
        <h3 class="font-semibold text-md mb-2 text-gray-400">Application Logs</h3>
        <LogViewer />
    </div>
</div>