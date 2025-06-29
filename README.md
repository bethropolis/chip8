# CHIP-8 Emulator

This is a CHIP-8 emulator built with Wails (Go backend) and Svelte (frontend) with Tailwind CSS v4 for styling.

## Getting Started

To run the emulator, follow these steps:

1.  **Navigate to the project directory:**
    ```bash
    cd chip8-wails
    ```

2.  **Run in development mode:**
    ```bash
    wails dev
    ```

    This will compile the Go backend, install frontend dependencies, and start the development server. The emulator UI should open in a new window.

## Usage

1.  **Load a ROM:** Click the "Load ROM" button in the emulator UI and select a CHIP-8 ROM file (typically with a `.ch8` extension).
2.  **Emulation:** The emulator will automatically start executing the loaded ROM. You should see the display update and debug information (registers, PC, etc.) in real-time.

## Project Structure

-   `app.go`: Wails application binding and main emulator loop.
-   `main.go`: Entry point for the Wails application.
-   `wails.json`: Wails configuration.
-   `chip8/`: Contains the core CHIP-8 CPU emulator logic (`chip8.go`) and unit tests (`chip8_test.go`).
-   `frontend/`: Svelte frontend application.
    -   `src/App.svelte`: Main UI component, including the display, controls, and debug panels.
    -   `src/app.css`: Tailwind CSS imports.
    -   `src/main.js`: Frontend entry point.
    -   `tailwind.config.js`: Tailwind CSS configuration.
    -   `postcss.config.js`: PostCSS configuration.

## Development

### Go Backend

To run Go tests for the CHIP-8 core:

```bash
cd chip8-wails
go test -v ./chip8
```

### Frontend

To build the frontend separately:

```bash
cd chip8-wails/frontend
bun run build
```

## Troubleshooting

If you encounter issues, try cleaning the build artifacts and regenerating Wails bindings:

```bash
cd chip8-wails
rm -rf frontend/node_modules frontend/dist frontend/src/wailsjs
wails dev
```
