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

Each parameter is applied per-direction (egress + ingress), so effective RTT is roughly 2x the delay value. Asymmetric values show ▲ upload and ▼ download on separate lines.

| Profile | Delay | Jitter | Dist | Loss | Reorder | Slot | Rate |
|---------|-------|--------|------|------|---------|------|------|
| 3G | 100ms | ▲ 50ms<br>▼ 30ms | paretonormal | ▲ 2.5%<br>▼ 1.5% | – | 40ms 10ms | ▲ 0.5 Mbit<br>▼ 2 Mbit |
| LTE | 20ms | ▲ 8ms<br>▼ 5ms | paretonormal | ▲ 1%<br>▼ 0.5% | – | 10ms 3ms | ▲ 15 Mbit<br>▼ 50 Mbit |
| 5G | 5ms | 1ms | paretonormal | ▲ 0.1%<br>▼ 0.05% | – | – | ▲ 100 Mbit<br>▼ 300 Mbit |
| Edge-2G | 150ms | ▲ 100ms<br>▼ 60ms | paretonormal | ▲ 8%<br>▼ 5% | – | 80ms 20ms | ▲ 0.05 Mbit<br>▼ 0.1 Mbit |
| Lossy-WiFi | 5ms | 3ms | pareto | 3% | 1% gap 5 | 5ms 2ms | 20 Mbit |
| Starlink | 20ms | ▲ 10ms<br>▼ 5ms | normal | ▲ 1%<br>▼ 0.5% | 0.5% | – | ▲ 20 Mbit<br>▼ 100 Mbit |
| Satellite | 300ms | ▲ 50ms<br>▼ 30ms | normal | ▲ 2.5%<br>▼ 1.5% | – | – | ▲ 1 Mbit<br>▼ 5 Mbit |
| DSL | 15ms | 3ms | normal | 0.2% | – | – | ▲ 3 Mbit<br>▼ 25 Mbit |
| Cable | 5ms | 1ms | normal | 0.05% | – | – | ▲ 20 Mbit<br>▼ 200 Mbit |
| Airplane-WiFi | 150ms | ▲ 50ms<br>▼ 30ms | pareto | ▲ 5%<br>▼ 3% | 1% gap 5 | 30ms 10ms | ▲ 1 Mbit<br>▼ 2 Mbit |
| Congested | 50ms | 40ms | paretonormal | 5% | 2% gap 3 | – | ▲ 0.5 Mbit<br>▼ 1 Mbit |
| Bursty | 10ms | 2ms | – | gemodel (burst) | – | – | 50 Mbit |
| ECN-Datacenter | 1ms | 0.5ms | normal | 2% ecn | – | – | 1 Gbit |
| ECN-WAN | 25ms | 5ms | normal | 0.5% ecn | – | – | ▲ 50 Mbit<br>▼ 100 Mbit |

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
| `ecn` | Mark packets with ECN CE bit instead of dropping (see below) | `true` |
| `duplicate` | Packet duplication probability | `0.5%` |
| `reorder` | Packet reordering — random or with gap (see below) | `1%` or `1% gap 5` |
| `corrupt` | Packet corruption probability | `0.1%` |
| `rate` | Bandwidth limit | `2mbit` |
| `slot` | Packet batching interval — holds then releases in bursts | `20ms 5ms` |

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

### ECN marking

When `ecn: true` is set alongside `loss`, packets are marked with the ECN CE (Congestion Experienced) bit instead of being dropped. The packet still arrives, but ECN-aware TCP stacks treat it as a congestion signal and slow down. Useful for testing DCTCP, BBR, or QUIC ECN behavior:

```yaml
profiles:
  My-ECN-Test:
    delay: 5ms
    loss: 1%
    ecn: true
    rate: 1gbit
```

### Reorder with gap

By default, `reorder` randomly reorders packets. Adding `gap N` makes it deterministic: every Nth packet is reordered with the given probability. This is more realistic for triggering TCP fast-retransmit (which fires after 3 duplicate ACKs):

```yaml
reorder: "1% gap 5"    # every 5th packet has a 1% chance of being reordered
```

### Slot-based emission

The `slot` parameter batches packets into time slots instead of sending them individually. Packets are held and released in bursts, simulating WiFi TDMA scheduling or cellular resource allocation:

```yaml
slot: "20ms 5ms"    # release a batch every 20ms ± 5ms jitter
```

This is especially noticeable for interactive traffic (VoIP, gaming) where micro-bursts affect perceived quality even when average throughput is fine.

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
