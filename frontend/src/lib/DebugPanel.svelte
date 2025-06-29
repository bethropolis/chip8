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
        const data = await GetMemory(memoryOffset, memoryLimit);
        // data is base64, needs decoding
        memoryData = new Uint8Array(atob(data).split('').map(char => char.charCodeAt(0)));
    }

    onMount(() => {
        // Fetch memory periodically for the viewer
        memoryUpdateInterval = setInterval(fetchMemoryView, 200);
        // Debug state is now pushed from App.svelte, no need for EventsOn here.
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
        // Simple scroll: just move by a fixed amount
        if (event.deltaY > 0) {
            memoryOffset += 16;
        } else {
            memoryOffset -= 16;
        }

        if (memoryOffset < 0) memoryOffset = 0;
        if (memoryOffset > 4096 - memoryLimit) memoryOffset = 4096 - memoryLimit;

        // Prevent page scroll
        event.preventDefault();
        fetchMemoryView(); // Fetch new view on scroll
    }

    async function toggleBreakpoint(address) {
        if (debugState.Breakpoints && debugState.Breakpoints[address]) {
            await ClearBreakpoint(address);
        } else {
            await SetBreakpoint(address);
        }
    }
</script>

<div class="grid grid-cols-1 md:grid-cols-3 gap-4 p-4 overflow-y-auto h-full">
    <!-- CPU Registers -->
    <div class="p-2 bg-[#34495e] rounded-lg border border-gray-700">
        <h3 class="font-bold text-lg mb-2">CPU Registers</h3>
        <div class="grid grid-cols-2 gap-x-4 text-sm font-mono">
            {#each { length: 8 } as _, i}
                <span>V{i.toString(16).toUpperCase()}: {`0x${debugState.Registers?.[i]?.toString(16).padStart(2, "0").toUpperCase() ?? "00"}`}</span>
                <span>V{(i + 8).toString(16).toUpperCase()}: {`0x${debugState.Registers?.[i + 8]?.toString(16).padStart(2, "0").toUpperCase() ?? "00"}`}</span>
            {/each}
        </div>
    </div>

    <!-- System State -->
    <div class="p-2 bg-[#34495e] rounded-lg border border-gray-700">
        <h3 class="font-bold text-lg mb-2">System State</h3>
        <div class="text-sm font-mono">
            <p>PC: {`0x${debugState.PC?.toString(16).padStart(4, "0").toUpperCase() ?? "0000"}`}</p>
            <p>I: {`0x${debugState.I?.toString(16).padStart(4, "0").toUpperCase() ?? "0000"}`}</p>
            <p>SP: {`0x${debugState.SP?.toString(16).padStart(2, "0").toUpperCase() ?? "00"}`}</p>
            <p>Delay Timer: {debugState.DelayTimer ?? "0"}</p>
            <p>Sound Timer: {debugState.SoundTimer ?? "0"}</p>
        </div>
    </div>

    <!-- Stack -->
    <div class="p-2 bg-[#34495e] rounded-lg border border-gray-700">
        <h3 class="font-bold text-lg mb-2">Stack</h3>
        <pre class="text-sm overflow-y-auto h-24 bg-slate-800 p-2 rounded-md border border-slate-700 font-mono">
            {#each debugState.Stack || [] as value, i}
                <div class:text-cyan-400={i === (debugState.SP > 0 ? debugState.SP -1 : 0)}>Stack[{i.toString(16).toUpperCase()}]: 0x{value.toString(16).padStart(4, "0").toUpperCase()}</div>
            {/each}
        </pre>
    </div>

    <!-- Disassembly -->
    <div class="p-2 bg-[#34495e] rounded-lg border border-gray-700 md:col-span-1">
        <h3 class="font-bold text-lg mb-2">Disassembly</h3>
        <pre class="text-xs leading-tight overflow-y-auto bg-slate-800 p-2 rounded-md border border-slate-700 h-64 font-mono">
            {#each debugState.Disassembly || [] as line}
                {@const address = parseInt(line.split(":")[0].replace("► ", ""), 16)}
                <div
                    class:text-cyan-400={line.startsWith("►")}
                    class:bg-red-700={debugState.Breakpoints && debugState.Breakpoints[address]}
                    on:click={() => toggleBreakpoint(address)}
                    class="cursor-pointer hover:bg-gray-600"
                >{line}</div>
            {/each}
        </pre>
    </div>

    <!-- Memory Viewer -->
    <div class="p-2 bg-[#34495e] rounded-lg border border-gray-700 md:col-span-2">
        <h3 class="font-bold text-lg mb-2">Memory Viewer (scroll to navigate)</h3>
        <div class="text-sm overflow-hidden h-64 bg-slate-800 p-2 rounded-md border border-slate-700 font-mono" on:wheel={handleMemoryScroll}>
            {#each Array(Math.ceil(memoryData.length / 16)) as _, rowIdx}
                <div class="flex">
                    <span class="text-gray-500 mr-2">{formatAddress(memoryOffset + rowIdx * 16)}:</span>
                    {#each Array(16) as _, colIdx}
                        {@const byte = memoryData[rowIdx * 16 + colIdx]}
                        <span class="mr-1">{byte !== undefined ? formatByte(byte) : "--"}</span>
                    {/each}
                </div>
            {/each}
        </div>
    </div>

    <!-- Logs -->
    <div class="p-2 bg-[#34495e] rounded-lg border border-gray-700 col-span-3">
        <h3 class="font-bold text-lg mb-2">Application Logs</h3>
        <LogViewer />
    </div>
</div>