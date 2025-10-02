#!/bin/bash

DROPFILEPATH=%1

cd /sbbs/xtrn2/history
./history -path $DROPFILEPATH

# optionally, specify:
#   cache-ttl (default 30m)
#   bypass-cache (default false)
# e.g.
#   ./history -path /sbbs/node1 -cache-ttl 1h -bypass-cache

