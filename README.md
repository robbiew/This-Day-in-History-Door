# This Day in History

BBS door written in Go that fetches "On This Day" events and displays them in an ANSI terminal suitable for BBS door usage. The original Mystic BBS mod and artwork was from by Smooth from the scene/modding group "PHENOM PRODUCTIONS." I've re-created this as a generic linux door program with a new data source (Wikimedia API). 

## Features

- ANSI/terminal styled output with color and simple layout
- Fetches historical events from the Wikimedia "On this day" API
- Optionally caches the data to make it more snappy (see command line options)
- Fits output into typical BBS screen area (80x24)
- Automatically exits after 2 minutes with no user input

## Requirements

- Go 1.21+ to build
- Internet access for Wikimedia API requests
- A door drop directory containing `door32.sys` (the program reads `door32.sys` from the provided `-path`)
- A Linux-based BBS (Mystic, Synchronet, Enigma 1/2, etc.)
- Users must be using a terminal program that supports ANSI/CP437 - there is no ascii fallback

## Building

1. **[Install Go](https://go.dev/doc/install) (version 1.21 or newer) if you haven't already.** 

2. **Clone the repository:**
   ```sh
   git clone https://github.com/robbiew/This-Day-in-History-Door.git
   cd This-Day-in-History-Door
   ```
   
3. **Download dependencies:**  
   ```sh
   go mod tidy
   ```

4. **Build the project:**
   ```sh
   go build -o history .
   ```
   This creates the executable named "history".


## Running

The program expects a `-path` to a BBS node directory that contains `door32.sys`. It can be a direct link to the dropfile or just the the path to the folder. Example:

```sh
./history -path /sbbs/node1
```

The included wrapper [`start.sh`](start.sh:1) shows how some setups might invoke the program. On a multi-node BBS, you'd pass the path the to dropfile as an argument:

```sh
#!/bin/bash
cd /sbbs/xtrn/history
./history -path %1
```

Command line flags (caching, selection, and display):

- `-bypass-cache` (boolean): force a fresh network fetch and ignore any valid cached response. Useful for debugging.
- `-cache-ttl` (duration): set the TTL used for on-disk cache entries. Accepts Go duration strings (e.g., `1h`, `30m`, `24h`). Default: `24h`.
- `-shuffle` (boolean, default: true): randomize both which events are selected and the order they are displayed. Enabled by default to increase visible variety between runs.
- `-strategy` (string): selection strategy to choose which events to display. Supported values:
  - `era-based` (default) — attempt to pick a small quota from each historical era (Ancient, Medieval, Early Modern, Modern, Contemporary), then fill remaining slots randomly.
  - `random` — random selection of events.
  - `oldest-first` — choose the oldest events (use `-shuffle=false` for deterministic oldest-first output).

How `-shuffle` and `-strategy` interact:
- Used together (recommended for variety): choose a strategy with `-strategy` and enable `-shuffle` (default). The program will select events according to the strategy and then apply randomness to selection and final ordering so repeated runs produce different, varied outputs.
  - Example: `./history -path /sbbs/node1 -strategy=era-based -shuffle`
- Used separately:
  - If you want a deterministic selection (no randomness), set `-shuffle=false` and pick a strategy:
    - `-strategy=oldest-first -shuffle=false` produces deterministic oldest-first output.
    - Note: `era-based` still uses internal selection quotas; if you require completely deterministic era-based selection I can add a deterministic mode.
  - If you only want random ordering of a fixed selection (not currently separate), set `-shuffle=true` (current implementation randomizes both selection and order). If you need independent control of selection vs ordering I can add separate flags (`-shuffle-selection` and `-shuffle-order`).

Examples:

```sh
# Default: era-based strategy with shuffle (varied selection + order)
./history -path /sbbs/node1

# Deterministic oldest-first (no selection randomness)
./history -path /sbbs/node1 -strategy=oldest-first -shuffle=false

# Random selection strategy with shuffle (random subset + random order)
./history -path /sbbs/node1 -strategy=random -shuffle

# Force fresh fetch and varied display
./history -path /sbbs/node1 -bypass-cache -shuffle

# Set cache TTL to 1 hour
./history -path /sbbs/node1 -cache-ttl 1h
```

Notes:
- `-shuffle` affects both selection and ordering. 
- The `-bypass-cache` flag prevents the client from writing the fetched response to disk (it fetches fresh data but leaves the on-disk cache unchanged).


## API and network behavior

- The program uses the Wikimedia feed endpoint (en.wikipedia.org) and sets a 10s HTTP timeout.
- If the API is unreachable the program prints an error message in the terminal.
