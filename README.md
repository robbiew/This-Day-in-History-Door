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

Command Line flags related to caching:

- `-bypass-cache` (boolean): force a fresh network fetch and ignore any valid cached response. Useful for debugging.
- `-cache-ttl` (duration): set the TTL used for on-disk cache entries. Accepts Go duration strings (e.g., `1h`, `30m`, `24h`). Default: `24h`.

Examples:

```sh
# Force a fresh fetch (ignore cache, fetches new data every load)
./history -path /sbbs/node1 -bypass-cache

# Set cache TTL to 1 hour (cache will be considered stale after 1h)
./history -path /sbbs/node1 -cache-ttl 1h

```


## API and network behavior

- The program uses the Wikimedia feed endpoint (en.wikipedia.org) and sets a 10s HTTP timeout.
- If the API is unreachable the program prints an error message in the terminal.
