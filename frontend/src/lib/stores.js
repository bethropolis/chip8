import { writable } from "svelte/store";
import { SaveSettings } from "../wailsjs/go/main/App.js";

/**
 * Svelte store for notification state.
 * @type {import("svelte/store").Writable<{message: string, type: string, show: boolean}>}
 */
export const notification = writable({
  message: "",
  type: "info",
  show: false,
});

/**
 * Show a notification with a message, type, and duration.
 * @param {string} message
 * @param {string} [type="info"]
 * @param {number} [duration=3000]
 */
export function showNotification(message, type = "info", duration = 3000) {
  notification.set({ message, type, show: true });
  setTimeout(() => {
    notification.update((n) => ({ ...n, show: false }));
  }, duration);
}

/**
 * Default emulator settings.
 * @type {{
 *   clockSpeed: number,
 *   displayColor: string,
 *   scanlineEffect: boolean,
 *   keyMap: Record<string|number, number>
 * }}
 */
const defaultSettings = {
  clockSpeed: 700,
  displayColor: "#33FF00",
  scanlineEffect: false,
  keyMap: {
    1: 0x1,
    2: 0x2,
    3: 0x3,
    4: 0xc,
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
  },
};

/**
 * Svelte store for emulator settings.
 * @type {import("svelte/store").Writable<typeof defaultSettings>}
 */
export const settings = writable(defaultSettings);

/**
 * Save settings to both the store and the Go backend.
 * @param {typeof defaultSettings} newSettings
 * @returns {Promise<void>}
 */
export async function updateAndSaveSettings(newSettings) {
  try {
    await SaveSettings(newSettings);
    settings.set(newSettings);
    showNotification("Settings saved successfully!", "success");
  } catch (error) {
    showNotification(`Failed to save settings: ${error}`, "error");
    console.error("Settings save error:", error);
  }
}

/**
 * Initializes the settings store.
 * If initialSettings are provided (from backend), they are used.
 * Otherwise, default settings are applied.
 * @param {object | null} initialSettings - Settings fetched from the backend, or null.
 */
export function initializeSettings(initialSettings) {
  console.log("Initializing settings with: ", initialSettings);
  settings.update((currentSettings) => {
    // Start with current settings (which are defaults if not yet loaded)
    const mergedSettings = { ...currentSettings };

    if (initialSettings) {
      // Merge top-level properties
      Object.assign(mergedSettings, initialSettings);

      // Deep merge for keyMap
      if (initialSettings.keyMap) {
        mergedSettings.keyMap = {
          ...currentSettings.keyMap,
          ...initialSettings.keyMap,
        };
      }
    }
    return mergedSettings;
  });
}
