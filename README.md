# lagsim

![lagsim](doc/header.png)

Network condition simulator for Linux routers. Injects latency, jitter, packet loss, reordering, and duplication per client IP using `tc`/`netem`/`ifb`.

Comes with built-in profiles for common network conditions (3G, LTE, Satellite, Starlink, etc.) and an interactive TUI to manage them. Profiles support asymmetric upload/download parameters to model real-world links.

![Screenshot](doc/screenshot.png)

## How it works

lagsim sets up an HTB qdisc tree on your LAN interface with per-client classes and netem leaf qdiscs. Ingress traffic is redirected through an IFB device so both upload and download are conditioned independently.

```
LAN clients <──eth0──> router <──wan0──> internet
                 │
         HTB + netem (egress/download)
         IFB + netem (ingress/upload)
```

Each direction gets its own netem parameters, so profiles can model asymmetric links (e.g., DSL with fast download / slow upload, or cellular with higher uplink loss).

## Install

```bash
go build -o lagsim .
sudo cp lagsim /usr/local/bin/
```

## Usage

### Interactive TUI

```bash
sudo lagsim
```

| Key | Action |
|-----|--------|
| `↑`/`k`, `↓`/`j` | Navigate client list |
| `Enter` | Open profile selection menu |
| `e` | Edit device name (requires MAC) |
| `r`, `Delete` | Remove profile from client |
| `Esc` | Cancel / back to list |
| `Ctrl+U` | Clear name in edit mode |
| `q`, `Ctrl+C` | Quit |

### CLI

```bash
# List clients and their profiles
sudo lagsim list

# Show available profiles
sudo lagsim profiles

# Apply a profile to a client
sudo lagsim apply 192.168.1.100 3G

# Remove conditioning from a client
sudo lagsim remove 192.168.1.100

# Initialize tc infrastructure and restore saved assignments
sudo lagsim init

# Tear down all tc rules
sudo lagsim teardown

# Dump raw tc state for debugging
sudo lagsim status
```

### Flags

| Flag | Description |
|------|-------------|
| `-c`, `--config` | Config file path (default `~/.config/lagsim.yaml`) |
| `--dry-run` | Print tc commands without executing |
| `-v`, `--verbose` | Verbose output |

On first run, lagsim auto-detects the LAN interface and subnet. If multiple interfaces are found, it prompts you to choose.

## Built-in profiles

Each parameter is applied per-direction (egress + ingress), so effective RTT is roughly 2x the delay value. Asymmetric values are shown as `▼ download ▲ upload`.

| Profile | Delay | Jitter | Dist | Loss | Reorder | Rate |
|---------|-------|--------|------|------|---------|------|
| 3G | 100ms | ▼ 30ms ▲ 50ms | paretonormal | ▼ 1.5% ▲ 2.5% | – | ▼ 2 Mbit ▲ 0.5 Mbit |
| LTE | 20ms | ▼ 5ms ▲ 8ms | paretonormal | ▼ 0.5% ▲ 1% | – | ▼ 50 Mbit ▲ 15 Mbit |
| 5G | 5ms | 1ms | paretonormal | ▼ 0.05% ▲ 0.1% | – | ▼ 300 Mbit ▲ 100 Mbit |
| Edge-2G | 150ms | ▼ 60ms ▲ 100ms | paretonormal | ▼ 5% ▲ 8% | – | ▼ 0.1 Mbit ▲ 0.05 Mbit |
| Lossy-WiFi | 5ms | 3ms | pareto | 3% | 1% | 20 Mbit |
| Starlink | 20ms | ▼ 5ms ▲ 10ms | normal | ▼ 0.5% ▲ 1% | 0.5% | ▼ 100 Mbit ▲ 20 Mbit |
| Satellite | 300ms | ▼ 30ms ▲ 50ms | normal | ▼ 1.5% ▲ 2.5% | – | ▼ 5 Mbit ▲ 1 Mbit |
| DSL | 15ms | 3ms | normal | 0.2% | – | ▼ 25 Mbit ▲ 3 Mbit |
| Cable | 5ms | 1ms | normal | 0.05% | – | ▼ 200 Mbit ▲ 20 Mbit |
| Airplane-WiFi | 150ms | ▼ 30ms ▲ 50ms | pareto | ▼ 3% ▲ 5% | 1% | ▼ 2 Mbit ▲ 1 Mbit |
| Congested | 50ms | 40ms | paretonormal | 5% | 2% | ▼ 1 Mbit ▲ 0.5 Mbit |
| Bursty | 10ms | 2ms | – | gemodel (burst) | – | 50 Mbit |

Built-in profiles are defined in code, not written to the config file.

## Configuration

Configuration is stored in `~/.config/lagsim.yaml`:

```yaml
interfaces:
  lan: eth0       # LAN-facing interface (auto-detected on first run)
  ifb: ifb0       # IFB device (created automatically)
  subnet: 192.168.1.0/24
root_rate: 1gbit

profiles:
  # Override a built-in profile
  3G:
    delay: 100ms
    jitter: 30ms
    correlation: 25%
    loss: 1.5%
    rate: 2mbit
    upload:
      rate: 0.5mbit

  # Add a custom profile
  My-VPN:
    delay: 30ms
    jitter: 5ms
    loss: 0.1%
    rate: 50mbit

  # Disable a built-in profile
  Edge-2G: null

assignments:
  192.168.1.100: 3G
  192.168.1.101: LTE

names:
  aa:bb:cc:dd:ee:f0: Living Room TV
  aa:bb:cc:dd:ee:f1: Dad's Phone
```

### Custom profiles

Only profiles that differ from the built-in defaults are saved to the config file. You can:

- **Add** custom profiles alongside the built-ins
- **Override** a built-in by redefining it (full replacement, not merged)
- **Disable** a built-in by setting it to `null`

### Profile parameters

| Parameter | Description | Example |
|-----------|-------------|---------|
| `delay` | Base latency added to each packet | `100ms` |
| `jitter` | Random variation added to delay | `30ms` |
| `correlation` | How much each packet's delay correlates with the previous | `25%` |
| `distribution` | Jitter distribution: `normal`, `pareto`, or `paretonormal` | `paretonormal` |
| `loss` | Packet loss — random or bursty (see below) | `1.5%` |
| `duplicate` | Packet duplication probability | `0.5%` |
| `reorder` | Packet reordering probability | `1%` |
| `corrupt` | Packet corruption probability | `0.1%` |
| `rate` | Bandwidth limit | `2mbit` |

All parameters are optional except `delay`. Values use `tc`/`netem` syntax.

### Delay distribution

Without a distribution, jitter is uniformly random. Setting `distribution` shapes how jitter values are picked:

- **`normal`** — bell curve around the base delay. Good for stable links (DSL, cable, satellite) where variation is symmetric.
- **`pareto`** — heavy-tailed: most packets are near the base delay, but occasional packets get much larger spikes. Good for WiFi and other interference-prone links.
- **`paretonormal`** — blend of both: normal most of the time with pareto-like tail spikes. Good for cellular networks where handoffs and contention cause intermittent latency bursts.

### Bursty loss

The `loss` field supports netem's Gilbert-Elliott model for realistic bursty loss patterns — periods of clean transmission interrupted by short bursts of heavy packet loss:

```yaml
loss: "gemodel p r 1-h 1-k"
```

| Parameter | Meaning |
|-----------|---------|
| `p` | Probability of entering the bad (lossy) state |
| `r` | Probability of returning to the good state |
| `1-h` | Loss rate in the bad state (e.g., `100%` = total blackout) |
| `1-k` | Loss rate in the good state (e.g., `0%` = no baseline loss) |

Example: `loss: "gemodel 0.5% 15% 100% 0%"` — clean most of the time, with occasional short bursts (~7 packets) of 100% loss. This models WiFi interference, cellular handoffs, or buffer overflows.

### Asymmetric profiles

Base parameters apply to both directions. Add `download` and/or `upload` sections to override specific parameters per direction — only the fields you specify are overridden, the rest inherit from the base:

```yaml
profiles:
  My-Satellite:
    delay: 300ms
    jitter: 30ms
    loss: 1.5%
    rate: 5mbit
    upload:
      rate: 1mbit       # slower upload
      jitter: 50ms      # more jitter on uplink
      loss: 2.5%        # more loss on uplink
    download:
      rate: 10mbit      # faster download
```

### Device names

Custom names are keyed by MAC address so they follow the device across IP changes. Edit names in the TUI with `e`, or set them directly in the config under `names`.

### Persistence

Assignments persist across reboots. Run `lagsim init` at startup to restore them (e.g. via systemd or cron `@reboot`):

```bash
# crontab -e
@reboot /usr/local/bin/lagsim init
```

## Requirements

- Linux with `tc`, `ip`, and the `ifb` kernel module
- Root privileges
- Go 1.24+ to build

## License

MIT
