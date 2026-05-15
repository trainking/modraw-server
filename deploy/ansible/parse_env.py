#!/usr/bin/env python3
"""Parse a KEY=VALUE .env file and output JSON on stdout."""
import json
import sys

if len(sys.argv) != 2:
    print("Usage: python3 parse_env.py <.env file>", file=sys.stderr)
    sys.exit(1)

d = {}
with open(sys.argv[1]) as f:
    for line in f:
        line = line.strip()
        if not line or line.startswith("#"):
            continue
        if "=" not in line:
            continue
        k, v = line.split("=", 1)
        k = k.strip()
        v = v.strip().strip('"').strip("'")
        d[k] = v

print(json.dumps(d))
