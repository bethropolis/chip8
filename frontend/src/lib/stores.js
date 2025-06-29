// frontend/src/lib/stores.js
import { writable } from "svelte/store";

// A writable store for our notification.
// It will hold an object with the message and type.
export const notification = writable({
  message: "",
  type: "info",
  show: false,
});

/**
 * Helper function to easily show a notification.
 * @param {string} message The message to display.
 * @param {string} [type='info'] The type of notification (info, success, warning, error).
 * @param {number} [duration=3000] How long to show the notification in ms.
 */
export function showNotification(message, type = "info", duration = 3000) {
  notification.set({ message, type, show: true });

  // Automatically hide the notification after the duration
  setTimeout(() => {
    notification.update((n) => ({ ...n, show: false }));
  }, duration);
}
