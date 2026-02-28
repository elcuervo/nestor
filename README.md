```
                  _
  _ __   ___  ___| |_ ___  _ __
 | '_ \ / _ \/ __| __/ _ \| '__|
 | | | |  __/\__ \ || (_) | |
 |_| |_|\___||___/\__\___/|_|
     NEtwork Share via TOR
```

## Install

Download a release binary from the [releases page](https://github.com/elcuervo/nestor/releases).

## Build from source

```bash
make download-tor
make build
```

## Usage

```bash
cd /path/to/share
nestor
# Go to http://....onion
```

To forward a local port instead of sharing files:

```bash
nestor --port 8080
# or
nestor -p 8080
```
