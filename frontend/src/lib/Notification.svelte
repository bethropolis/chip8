<script>
    import { fade } from "svelte/transition";
    import { notification } from "./stores.js";

    let timeout;

    // Dismiss the notification and clear the timeout
    function dismiss() {
        clearTimeout(timeout);
        notification.set({ ...$notification, show: false });
    }

    let bgColorClass;
    $: {
        if ($notification.show && $notification.message) {
            clearTimeout(timeout);
            timeout = setTimeout(dismiss, 3000);
        }
        switch ($notification.type) {
            case "success":
                bgColorClass = "bg-green-500";
                break;
            case "warning":
                bgColorClass = "bg-yellow-500";
                break;
            case "error":
                bgColorClass = "bg-red-500";
                break;
            case "info":
            default:
                bgColorClass = "bg-blue-500";
        }
    }
</script>

{#if $notification.show && $notification.message}
    <div
        in:fade={{ duration: 150 }}
        out:fade={{ duration: 150 }}
        class="fixed bottom-4 right-4 p-4 rounded-lg shadow-lg text-white flex items-center space-x-3 z-50 {bgColorClass}"
        role="alert"
    >
        <span>{$notification.message}</span>
        <button
            on:click={dismiss}
            class="ml-auto text-white opacity-75 hover:opacity-100"
        >
            <svg
                xmlns="http://www.w3.org/2000/svg"
                class="h-5 w-5"
                viewBox="0 0 20 20"
                fill="currentColor"
            >
                <path
                    fill-rule="evenodd"
                    d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
                    clip-rule="evenodd"
                />
            </svg>
        </button>
    </div>
{/if}
