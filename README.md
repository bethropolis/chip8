# CHIP-8 Emulator

[![Build Status](https://github.com/bethropolis/chip8/actions/workflows/test-build.yml/badge.svg)](https://github.com/bethropolis/chip8/actions/workflows/test-build.yml) [![Go Version](https://img.shields.io/badge/Go-1.22-00ADD8?logo=go&logoColor=white)](https://go.dev/) [![Wails Version](https://img.shields.io/badge/Wails-v2-blueviolet?logo=wails&logoColor=white)](https://wails.io/) [![License](https://img.shields.io/github/license/bethropolis/chip8)](https://github.com/bethropolis/chip8/blob/main/LICENSE)


This is a feature-rich CHIP-8 emulator built with Wails (Go backend) and Svelte (frontend) with Tailwind CSS v4 for styling.

## Download

You can download the latest releases for your operating system from the [releases page](https://github.com/bethropolis/chip8/releases).

## Features

*   **Customizable Settings:** Adjust display color, enable/disable scanline effects, change CPU clock speed, and remap keyboard controls via an intuitive settings modal.
*   **Enhanced Application Menu:**
    *   **File:** Load ROMs via file dialog, Save/Load emulator state, Quit.
    *   **Emulation:** Pause/Resume, Soft Reset (reload current ROM), Hard Reset (clear all state).
    *   **Help:** Dynamic "About" dialog displaying application version and details, and a direct link to the GitHub repository.
*   **Interactive Keypad:** On-screen keypad for direct input.
*   **Debug Panel:** Real-time view of CPU registers, memory, stack, and disassembly.
*   **Screenshot & State Saving:** Capture screenshots and save/load emulator state to/from files.
*   **Gamepad Support:** Basic gamepad input mapping.
*   **Drag-and-Drop ROM Loading:** Easily load ROMs by dragging them onto the emulator window.

## Getting Started

To run the emulator, follow these steps:

1.  **Navigate to the project directory:**
    ```bash
    cd chip8
    ```

2.  **Run in development mode:**
    ```bash
    wails dev
    ```

    This will compile the Go backend, install frontend dependencies, and start the development server. The emulator UI should open in a new window.

## Usage

1.  **Load a ROM:** Use the "File" -> "Load ROM..." menu option or drag and drop a `.ch8` file onto the emulator window.
2.  **Emulation:** The emulator will automatically start executing the loaded ROM.
3.  **Controls:** Use the keyboard (remappable in settings) or the on-screen keypad.
4.  **Settings:** Access settings via the gear icon in the header or the application menu.
5.  **Debug:** Switch to the "Debug" tab to view internal emulator state.

## Project Structure

-   `app.go`: Wails application binding, main emulator loop, and backend logic for settings, file operations, etc.
-   `main.go`: Entry point for the Wails application, Wails configuration, and application menu definition.
-   `wails.json`: Wails project configuration, including application metadata (version, URL).
-   `chip8/`: Contains the core CHIP-8 CPU emulator logic (`chip8.go`) and unit tests (`chip8_test.go`).
-   `frontend/`: Svelte frontend application.
    -   `src/App.svelte`: Main UI component, orchestrating other Svelte components.
    -   `src/lib/`: Contains reusable Svelte components (e.g., `SettingsModal.svelte`, `EmulatorView.svelte`, `DebugPanel.svelte`, `stores.js`).
    -   `src/stores.js`: Svelte stores for managing global application state like settings and notifications.
    -   `src/app.css`: Tailwind CSS imports.
    -   `src/main.js`: Frontend entry point.
    -   `tailwind.config.js`: Tailwind CSS configuration.
    -   `postcss.config.js`: PostCSS configuration.
    -   `wailsjs/`: Auto-generated Wails bindings for Go backend communication.

## Development

### Go Backend

To run Go tests for the CHIP-8 core:

```bash
cd chip8
go test -v ./chip8
```

### Frontend

To install frontend dependencies:

```bash
cd frontend
bun install
```

To build the frontend separately:

```bash
cd frontend
bun run build
```

## Troubleshooting

If you encounter issues, try cleaning the build artifacts and regenerating Wails bindings:

```bash
cd chip8
rm -rf frontend/node_modules frontend/dist frontend/src/wailsjs
wails dev
```
