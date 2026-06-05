#!/usr/bin/env python3
import argparse
import asyncio
import json
import os
import random
import signal
import sys
import time
from datetime import datetime, timezone


DEVICE_TYPE_CYCLE = ("rectifier", "dc_switchgear")


def load_config_from_file(path):
    with open(path, "r", encoding="utf-8") as f:
        return json.load(f)


def build_device_registry(substations, devices_per_sub, lines):
    devices = []
    subs_per_line = substations // lines
    remainder = substations % lines

    for line_idx in range(1, lines + 1):
        line_id = f"L{line_idx}"
        line_subs = subs_per_line + (1 if line_idx <= remainder else 0)

        for sub_seq in range(1, line_subs + 1):
            sub_id = f"SUB_{line_id}_{sub_seq:02d}"
            devices.append((sub_id, "substation", line_id))

            for dev_seq in range(1, devices_per_sub + 1):
                dtype = DEVICE_TYPE_CYCLE[(dev_seq - 1) % len(DEVICE_TYPE_CYCLE)]
                dev_id = f"DEV_{line_id}_{sub_seq:02d}_{dev_seq:02d}"
                devices.append((dev_id, dtype, line_id))

    return devices


def generate_telemetry(device_id, device_type, cfg):
    if device_type == "substation":
        voltage = random.gauss(cfg["voltage_nominal"], cfg["voltage_stddev"])
        current = random.uniform(500, 1500)
        temp_base = 45
    elif device_type == "rectifier":
        voltage = random.gauss(cfg["voltage_nominal"], cfg["voltage_stddev"])
        current = random.uniform(200, 800)
        temp_base = 55
    else:
        voltage = random.gauss(cfg["voltage_nominal"], cfg["voltage_stddev"])
        current = random.uniform(100, 600)
        temp_base = 40

    voltage = max(cfg["voltage_nominal"] * 0.733, min(cfg["voltage_nominal"] * 1.067, voltage))
    current = max(10, current)

    if random.random() < cfg["voltage_dip_probability"]:
        voltage = random.uniform(
            cfg["voltage_nominal"] * 0.733,
            cfg["voltage_nominal"] * 0.833,
        )

    if random.random() < cfg["abnormal_probability"]:
        load_rate = random.uniform(cfg["abnormal_load_min"], cfg["abnormal_load_max"])
    else:
        load_rate = random.uniform(cfg["normal_load_min"], cfg["normal_load_max"])

    if load_rate > 100:
        current = current * (load_rate / 70.0)

    power = voltage * current
    temperature = temp_base + random.uniform(-10, 30)
    temperature = max(25, min(80, temperature))

    return {
        "device_id": device_id,
        "device_type": device_type,
        "voltage": round(voltage, 1),
        "current": round(current, 1),
        "power": round(power, 1),
        "temperature": round(temperature, 1),
        "load_rate": round(load_rate, 1),
        "timestamp": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
    }


async def send_batch(writer, devices, cfg):
    lines = []
    for device_id, device_type, _ in devices:
        t = generate_telemetry(device_id, device_type, cfg)
        lines.append(json.dumps(t, separators=(",", ":")))

    payload = "\n".join(lines) + "\n"
    data = payload.encode("utf-8")
    writer.write(data)
    await writer.drain()
    return len(lines)


def print_dry_run(devices, cfg):
    batch = []
    for device_id, device_type, _ in devices:
        t = generate_telemetry(device_id, device_type, cfg)
        batch.append(t)

    print(json.dumps(batch, indent=2, ensure_ascii=False))
    print(f"\n--- Dry Run Summary ---", file=sys.stderr)
    print(f"Total devices: {len(devices)}", file=sys.stderr)
    type_counts = {}
    for _, dt, _ in devices:
        type_counts[dt] = type_counts.get(dt, 0) + 1
    for dt, c in type_counts.items():
        print(f"  {dt}: {c}", file=sys.stderr)
    print(f"Sample batch size: {len(batch)}", file=sys.stderr)


async def run_simulator(host, port, interval, cfg, shutdown_event):
    devices = build_device_registry(
        cfg["substations"], cfg["devices_per_sub"], cfg["lines"]
    )
    print(f"Device registry built: {len(devices)} devices")
    type_counts = {}
    for _, dt, _ in devices:
        type_counts[dt] = type_counts.get(dt, 0) + 1
    for dt, c in type_counts.items():
        print(f"  {dt}: {c}")

    total_sent = 0
    start_time = time.monotonic()
    last_stat_time = start_time

    while not shutdown_event.is_set():
        try:
            reader, writer = await asyncio.open_connection(host, port)
            peer = writer.get_extra_info("peername")
            print(f"Connected to {peer}")

            try:
                while not shutdown_event.is_set():
                    batch_size = await send_batch(writer, devices, cfg)
                    total_sent += batch_size

                    now = time.monotonic()
                    elapsed = now - last_stat_time
                    if elapsed >= 10.0:
                        rate = total_sent / (now - start_time)
                        print(
                            f"[Stats] Total sent: {total_sent}, "
                            f"Rate: {rate:.1f} msgs/sec, "
                            f"Uptime: {now - start_time:.0f}s"
                        )
                        last_stat_time = now

                    try:
                        await asyncio.wait_for(
                            shutdown_event.wait(), timeout=interval
                        )
                    except asyncio.TimeoutError:
                        pass
            except (ConnectionResetError, BrokenPipeError, ConnectionAbortedError) as e:
                print(f"Connection lost: {e}")
            finally:
                writer.close()
                try:
                    await writer.wait_closed()
                except Exception:
                    pass

        except (ConnectionRefusedError, OSError) as e:
            if shutdown_event.is_set():
                break
            print(f"Cannot connect to {host}:{port} - {e}, retrying in 3s...")
            try:
                await asyncio.wait_for(shutdown_event.wait(), timeout=3.0)
            except asyncio.TimeoutError:
                pass

    elapsed = time.monotonic() - start_time
    rate = total_sent / elapsed if elapsed > 0 else 0
    print(f"\nShutdown complete. Total sent: {total_sent}, Avg rate: {rate:.1f} msgs/sec, Uptime: {elapsed:.0f}s")


def parse_args():
    parser = argparse.ArgumentParser(
        description="IEC 61850 Simulator - sends telemetry data to Go backend via TCP"
    )
    parser.add_argument(
        "--host", default=None, help="Go backend TCP host (default: localhost)"
    )
    parser.add_argument(
        "--port", type=int, default=None, help="Go backend TCP port (default: 61850)"
    )
    parser.add_argument(
        "--interval",
        type=float,
        default=None,
        help="Send interval in seconds (default: 1.0)",
    )
    parser.add_argument(
        "--substations",
        type=int,
        default=None,
        help="Number of substations to simulate (default: 60)",
    )
    parser.add_argument(
        "--devices-per-sub",
        type=int,
        default=None,
        help="Number of devices per substation (default: 10)",
    )
    parser.add_argument(
        "--lines",
        type=int,
        default=None,
        help="Number of metro lines (default: 3)",
    )
    parser.add_argument(
        "--fault-mode",
        action="store_true",
        help="Increase overload probability for alarm testing",
    )
    parser.add_argument(
        "--normal-load-min",
        type=float,
        default=None,
        help="Minimum normal load rate %% (default: 30)",
    )
    parser.add_argument(
        "--normal-load-max",
        type=float,
        default=None,
        help="Maximum normal load rate %% (default: 70)",
    )
    parser.add_argument(
        "--abnormal-load-min",
        type=float,
        default=None,
        help="Minimum abnormal load rate %% (default: 105)",
    )
    parser.add_argument(
        "--abnormal-load-max",
        type=float,
        default=None,
        help="Maximum abnormal load rate %% (default: 120)",
    )
    parser.add_argument(
        "--abnormal-probability",
        type=float,
        default=None,
        help="Probability of abnormal load 0.0-1.0 (default: 0.05)",
    )
    parser.add_argument(
        "--voltage-nominal",
        type=float,
        default=None,
        help="Nominal voltage in V (default: 1500)",
    )
    parser.add_argument(
        "--voltage-stddev",
        type=float,
        default=None,
        help="Voltage standard deviation (default: 25)",
    )
    parser.add_argument(
        "--voltage-dip-probability",
        type=float,
        default=None,
        help="Probability of voltage dip (default: 0.02)",
    )
    parser.add_argument(
        "--config",
        default=None,
        help="Path to JSON config file that overrides CLI defaults",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Print first batch of telemetry to stdout without connecting",
    )
    return parser.parse_args()


def build_config(args):
    defaults = {
        "substations": 60,
        "devices_per_sub": 10,
        "lines": 3,
        "interval": 1.0,
        "normal_load_min": 30.0,
        "normal_load_max": 70.0,
        "abnormal_load_min": 105.0,
        "abnormal_load_max": 120.0,
        "abnormal_probability": 0.05,
        "voltage_nominal": 1500.0,
        "voltage_stddev": 25.0,
        "voltage_dip_probability": 0.02,
    }

    cfg = dict(defaults)

    if args.config:
        file_cfg = load_config_from_file(args.config)
        for key in defaults:
            if key in file_cfg:
                cfg[key] = file_cfg[key]

    cli_map = {
        "substations": args.substations,
        "devices_per_sub": args.devices_per_sub,
        "lines": args.lines,
        "interval": args.interval,
        "normal_load_min": args.normal_load_min,
        "normal_load_max": args.normal_load_max,
        "abnormal_load_min": args.abnormal_load_min,
        "abnormal_load_max": args.abnormal_load_max,
        "abnormal_probability": args.abnormal_probability,
        "voltage_nominal": args.voltage_nominal,
        "voltage_stddev": args.voltage_stddev,
        "voltage_dip_probability": args.voltage_dip_probability,
    }
    for key, val in cli_map.items():
        if val is not None:
            cfg[key] = val

    if args.fault_mode:
        cfg["abnormal_probability"] = max(cfg["abnormal_probability"], 0.20)

    return cfg


def main():
    args = parse_args()
    cfg = build_config(args)

    host = args.host or "localhost"
    port = args.port or 61850
    interval = cfg["interval"]

    print(f"=== IEC 61850 Simulator ===")
    print(f"Target: {host}:{port}")
    print(f"Interval: {interval}s")
    print(f"Substations: {cfg['substations']}")
    print(f"Devices per substation: {cfg['devices_per_sub']}")
    print(f"Lines: {cfg['lines']}")
    print(f"Fault mode: {'ON' if args.fault_mode else 'OFF'}")
    print(f"Normal load: {cfg['normal_load_min']}%-{cfg['normal_load_max']}%")
    print(f"Abnormal load: {cfg['abnormal_load_min']}%-{cfg['abnormal_load_max']}%")
    print(f"Abnormal probability: {cfg['abnormal_probability']:.2f}")
    print(f"Voltage nominal: {cfg['voltage_nominal']}V, stddev: {cfg['voltage_stddev']}")
    print(f"Voltage dip probability: {cfg['voltage_dip_probability']:.2f}")
    total_devices = cfg["substations"] + cfg["substations"] * cfg["devices_per_sub"]
    print(f"Total devices: {total_devices}")
    print()

    if args.dry_run:
        devices = build_device_registry(
            cfg["substations"], cfg["devices_per_sub"], cfg["lines"]
        )
        print_dry_run(devices, cfg)
        return

    shutdown_event = asyncio.Event()

    def _signal_handler(sig, frame):
        print(f"\nReceived signal {sig}, shutting down...")
        shutdown_event.set()

    signal.signal(signal.SIGINT, _signal_handler)
    signal.signal(signal.SIGTERM, _signal_handler)

    try:
        asyncio.run(run_simulator(host, port, interval, cfg, shutdown_event))
    except KeyboardInterrupt:
        print("\nSimulator stopped.")


if __name__ == "__main__":
    main()
