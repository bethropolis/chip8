# CHIP-8 Emulator Codebase Analysis Instructions for Gemini

## Project Overview
You are working with an existing CHIP-8 emulator built using Wails (Go backend + Svelte frontend). Your task is to understand, analyze, and work with the existing codebase rather than building from scratch.

## Understanding This Wails Project

### What is Wails?
Wails is a framework that allows you to build desktop applications using Go for the backend and web technologies (HTML/CSS/JavaScript) for the frontend. It creates a bridge between Go and the frontend, allowing:
- Go methods to be called from the frontend
- Events to be emitted from Go to the frontend
- Native OS dialogs and features

### Project Architecture
This CHIP-8 emulator consists of:
- **Go Backend**: CHIP-8 CPU emulation logic, ROM loading, file handling
- **Svelte Frontend**: User interface, display rendering, controls
- **Wails Bridge**: Connects Go backend to Svelte frontend

## Step-by-Step Codebase Analysis

### 1. First, Explore the Project Structure
Start by examining the project directory structure:
```bash
# List all files and directories
find . -type f -name "*.go" -o -name "*.svelte" -o -name "*.js" -o -name "*.json" | head -20
```

Look for these key files:
- `main.go` - Entry point, Wails app configuration
- `app.go` - Backend application logic and methods
- `wails.json` - Wails project configuration
- `go.mod` - Go module dependencies
- `frontend/` directory - Contains Svelte frontend
- `chip8/` directory - CHIP-8 emulator core logic

### 2. Understand the Go Backend
Examine the Go files to understand:

**Key Files to Analyze:**
- `main.go`: How the Wails app is initialized and configured
- `app.go`: The main application struct and methods exposed to frontend
- `chip8/chip8.go`: Core CHIP-8 emulator implementation

**Questions to Answer:**
- What struct represents the main application?
- What methods are exposed to the frontend?
- How is the CHIP-8 CPU core implemented?
- What events are emitted to the frontend?
- How are timers and the emulation loop handled?

### 3. Understand the Frontend
Examine the Svelte frontend:

**Key Files to Analyze:**
- `frontend/src/App.svelte`: Main UI component
- `frontend/src/main.js`: Frontend entry point
- `frontend/package.json`: Frontend dependencies
- `frontend/vite.config.js`: Build configuration

**Questions to Answer:**
- How is the UI structured and styled?
- How does it communicate with the Go backend?
- How are events from Go handled?
- How is the CHIP-8 display rendered?
- What controls and debug information are available?

### 4. Understand the Wails Integration
Examine how Go and Svelte communicate:

**Look for:**
- `wailsjs/` directory in frontend (auto-generated bindings)
- Import statements in Svelte files
- Event listeners and emitters
- Method calls between frontend and backend

### 5. Analyze the CHIP-8 Implementation
Understand the emulator core:

**Examine:**
- CHIP-8 CPU struct and fields
- Instruction decoding and execution
- Memory management
- Display rendering
- Input handling
- Timer implementation

## Key Concepts to Understand

### Wails Binding System
- Go methods can be called from frontend if they're on a bound struct
- Methods must be public (start with capital letter)
- Return values are passed back to frontend as promises

### Event System
- Go can emit events to frontend using `runtime.EventsEmit()`
- Frontend listens with `EventsOn()` from wails runtime
- Used for real-time updates (display, debug info, status)

### CHIP-8 Emulator Basics
- 4KB memory, 16 registers, 64x32 display
- Instruction set with ~35 opcodes
- 60Hz timer system
- Hexadecimal keypad input

## Analysis Tasks

### 1. Code Flow Analysis
Trace the execution flow:
1. How does the app start? (`main.go`)
2. How is the emulator initialized?
3. How does ROM loading work?
4. How does the emulation loop run?
5. How are display updates handled?

### 2. Interface Analysis
Understand the Go-Svelte interface:
1. What methods can Svelte call?
2. What events does Go emit?
3. How is data passed between them?
4. What's the timing and synchronization?

### 3. UI Analysis
Understand the user interface:
1. How is the layout structured?
2. How are controls implemented?
3. How is the CHIP-8 display rendered?
4. How is debug information shown?

### 4. Issue Identification
Look for potential issues:
1. Race conditions between Go and frontend
2. Performance bottlenecks
3. Memory leaks
4. UI responsiveness problems
5. Error handling gaps

## Working with the Codebase

### Development Commands
```bash
# Install frontend dependencies
cd frontend && npm install

# Run in development mode
wails dev

# Build for production
wails build

# Run Go tests
go test ./...
```

### Making Changes
1. **Backend changes**: Modify Go files, Wails will auto-regenerate bindings
2. **Frontend changes**: Modify Svelte files, Vite will hot-reload
3. **Adding methods**: Add public methods to app struct in `app.go`
4. **Adding events**: Use `runtime.EventsEmit()` in Go, `EventsOn()` in Svelte

### Debugging
1. Use browser dev tools for frontend debugging
2. Use Go debugging tools for backend
3. Check console for Wails-specific errors
4. Verify binding generation in `wailsjs/` directory

## Success Criteria

You should be able to:
1. Explain the overall architecture and data flow
2. Identify the main components and their responsibilities
3. Understand how Go and Svelte communicate
4. Locate and understand the CHIP-8 emulator logic
5. Run the application successfully
6. Make small modifications and see the results
7. Identify potential improvements or issues

## Next Steps

After understanding the codebase:
1. Test the current functionality
2. Identify what works and what doesn't
3. Look for bugs or missing features
4. Plan improvements or fixes
5. Understand the specific requirements for your tasks

Remember: This is a working codebase, not a tutorial. Focus on understanding the existing implementation rather than building from scratch.