```
                 _
░▒▓███████▓▒░░▒▓████████▓▒░░▒▓███████▓▒░▒▓████████▓▒░▒▓██████▓▒░░▒▓███████▓▒░
░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░      ░▒▓█▓▒░         ░▒▓█▓▒░  ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░
░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░      ░▒▓█▓▒░         ░▒▓█▓▒░  ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░
░▒▓█▓▒░░▒▓█▓▒░▒▓██████▓▒░  ░▒▓██████▓▒░   ░▒▓█▓▒░  ░▒▓█▓▒░░▒▓█▓▒░▒▓███████▓▒░
░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░             ░▒▓█▓▒░  ░▒▓█▓▒░  ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░
░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░             ░▒▓█▓▒░  ░▒▓█▓▒░  ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░
░▒▓█▓▒░░▒▓█▓▒░▒▓████████▓▒░▒▓███████▓▒░   ░▒▓█▓▒░   ░▒▓██████▓▒░░▒▓█▓▒░░▒▓█▓▒░

NEtwork Share via TOR

made with ☠️ by elcuervo
```

Like `python -m http.server`, but instead of binding to a local port it serves the current directory through a Tor hidden service. You get a `.onion` address you can hand to anyone without exposing your IP, opening firewall ports, or setting up any infrastructure.

It also works the other way: if you already have something running locally, you can forward that port through Tor instead of serving files.

Tor is bundled inside the binary, so there is nothing to install separately. Single binary, no runtime dependencies, works on macOS, Linux, and Windows.

## Install

**macOS / Linux**
```bash
curl -fsSL https://raw.githubusercontent.com/elcuervo/nestor/master/run.sh | sh
```

**Windows** (PowerShell)
```powershell
irm https://raw.githubusercontent.com/elcuervo/nestor/master/run.ps1 | iex
```

**Nix**
```bash
nix run github:elcuervo/nestor
```

Or download a binary directly from the [releases page](https://github.com/elcuervo/nestor/releases).

## Build from source

```bash
make download-tor
make build
```

Requires Go 1.25+. The `make download-tor` step fetches and embeds the Tor binary for your target platform before compiling.

If you use Nix, the included `flake.nix` sets up a dev shell with the right Go version and all required tools:

```bash
nix develop
make download-tor
make build
```

## Usage

Share the current directory:

```bash
cd /path/to/share
nestor
# Go to http://....onion
```

Forward a local port instead of serving files:

```bash
nestor --port 8080
# or
nestor -p 8080
```

Quiet mode prints only the onion URL and nothing else, which is useful for scripting or piping the address elsewhere:

```bash
nestor --quiet
nestor -q
```

## How it works

When you run nestor it starts a local HTTP file server (or uses the port you specified), then launches the embedded Tor binary and registers a hidden service pointed at that local address. Once Tor finishes bootstrapping the `.onion` address is printed and the service is live. Ctrl-C shuts everything down cleanly.

Tor is embedded at compile time using Go's `embed` package. That is what keeps the binary self-contained and avoids the need for a system Tor installation.
