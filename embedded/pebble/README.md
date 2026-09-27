# OCI On-Call Watch App (Pebble Alloy Embedded JS)

On-call alert response watch application for Orange Cat Investments (OCI), written in embedded JavaScript targeting the **RePebble Alloy (Moddable XS)** runtime with a small C bootstrap.

## Overview
This smartwatch application receives active IT tickets and on-call alerts pushed from `cmd/pebble-proxy` (`:8082`), displays high-priority infrastructure warnings, and sends physical button acknowledgement (ACK) signals back to the operations gateway.

## Project Structure
* `package.json`: Pebble project metadata (UUID, target platforms `emery` & `gabbro`, AppMessage dictionary mapping).
* `src/embeddedjs/main.js`: Watch-side UI rendering and button click event handlers.
* `src/embeddedjs/alertCore.js`: Pure platform-independent domain logic for alert queueing and state management.
* `src/embeddedjs/alertCore.test.js`: Unit tests for domain logic executed via `node --test`.
* `src/embeddedjs/manifest.json`: Moddable XS module manifest.
* `src/c/mdbl.c`: Minimal C bootstrap initializing the Moddable XS virtual machine.
* `src/pkjs/index.js`: PebbleKit JS phone-side proxy polling `cmd/pebble-proxy`.

## Building & Testing

```bash
# Run domain model unit tests
node --test embedded/pebble/src/embeddedjs/alertCore.test.js

# Build watch app for target platforms (RePebble Alloy SDK required)
pebble build

# Install on emulator
pebble install --emulator emery
```
