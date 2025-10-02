# history


BBS door written in Go that fetches "On This Day" events from Wikipedia and displays them in an ANSI terminal suitable for BBS door wrappers. Original artwork by Smooth from PHENOM.

## Features

- ANSI/terminal styled output with color and simple layout
- Fetches historical events from the Wikimedia "On this day" API
- Fits output into typical BBS screen area and supports basic terminal detection

## Requirements

- Go 1.17+ to build
- Internet access for Wikimedia API requests
- A door drop directory containing `door32.sys` (the program reads `door32.sys` from the provided `-path`)

## Building

```sh
go build -o history .
```

## Running

The program expects a path to a BBS node directory that contains `door32.sys`. It can be a direct link to the dropfile or just the the path to the folder. Example:

```sh
./history -path /sbbs/node1
```

The included wrapper [`start.sh`](start.sh:1) shows how some setups might invoke the program:

```sh
#!/bin/bash
cd /sbbs/xtrn/history
./history -path %1
```

CLI flags related to caching:

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
