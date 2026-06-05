#!/usr/bin/env python3
import argparse
import asyncio
import json
import random
import time
from datetime import datetime, timezone


DEVICE_TYPES = ("substation", "rectifier", "dc_switchgear")
LINE_IDS = ("L1", "L2", "L3")
RECT_PER_LINE = (67, 67, 66)
DCS_PER_LINE = (133, 134, 133)


def build_device_registry():
    devices = []
    for li, line_id in enumerate(LINE_IDS):
        for si in range(1, 21):
            sub_id = f"SUB_{line_id}_{si:02d}"
            devices.append((sub_id, "substation", line_id))

    for li, line_id in enumerate(LINE_IDS):
        count = RECT_PER_LINE[li]
        assigned = 0
        si = 1
        while assigned < count:
            per_sub = 4 if si <= (count - 60) else 3
            for ri in range(1, per_sub + 1):
                rect_id = f"RECT_{line_id}_{si:02d}_{ri:02d}"
                devices.append((rect_id, "rectifier", line_id))
                assigned += 1
                if assigned >= count:
                    break
            si += 1
            if si > 20:
                break

    for li, line_id in enumerate(LINE_IDS):
        count = DCS_PER_LINE[li]
        assigned = 0
        si = 1
        while assigned < count:
            per_sub = 7 if si <= (count - 120) else 6
            for di in range(1, per_sub + 1):
                dcs_id = f"DCS_{line_id}_{si:02d}_{di:02d}"
                devices.append((dcs_id, "dc_switchgear", line_id))
                assigned += 1
                if assigned >= count:
                    break
            si += 1
            if si > 20:
                break

    return devices


def generate_telemetry(device_id, device_type, fault_mode):
    if device_type == "substation":
        voltage = random.gauss(1500, 25)
        current = random.uniform(500, 1500)
        temp_base = 45
    elif device_type == "rectifier":
        voltage = random.gauss(1500, 25)
        current = random.uniform(200, 800)
        temp_base = 55
    else:
        voltage = random.gauss(1500, 25)
        current = random.uniform(100, 600)
        temp_base = 40

    voltage = max(1100, min(1600, voltage))
    current = max(10, current)

    if random.random() < 0.02:
        voltage = random.uniform(1100, 1250)

    overload_prob = 0.20 if fault_mode else 0.05
    if random.random() < overload_prob:
        load_rate = random.uniform(105, 120)
    else:
        load_rate = random.uniform(30, 70)

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


async def send_batch(writer, devices, fault_mode):
    lines = []
    for device_id, device_type, _ in devices:
        t = generate_telemetry(device_id, device_type, fault_mode)
        lines.append(json.dumps(t, separators=(",", ":")))

    payload = "\n".join(lines) + "\n"
    data = payload.encode("utf-8")
    writer.write(data)
    await writer.drain()
    return len(lines)


async def run_simulator(host, port, interval, fault_mode):
    devices = build_device_registry()
    print(f"Device registry built: {len(devices)} devices")
    type_counts = {}
    for _, dt, _ in devices:
        type_counts[dt] = type_counts.get(dt, 0) + 1
    for dt, c in type_counts.items():
        print(f"  {dt}: {c}")

    total_sent = 0
    start_time = time.monotonic()
    last_stat_time = start_time

    while True:
        try:
            reader, writer = await asyncio.open_connection(host, port)
            peer = writer.get_extra_info("peername")
            print(f"Connected to {peer}")

            try:
                while True:
                    batch_size = await send_batch(writer, devices, fault_mode)
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

                    await asyncio.sleep(interval)
            except (ConnectionResetError, BrokenPipeError, ConnectionAbortedError) as e:
                print(f"Connection lost: {e}")
            finally:
                writer.close()
                try:
                    await writer.wait_closed()
                except Exception:
                    pass

        except (ConnectionRefusedError, OSError) as e:
            print(f"Cannot connect to {host}:{port} - {e}, retrying in 3s...")
            await asyncio.sleep(3)


def main():
    parser = argparse.ArgumentParser(
        description="IEC 61850 Simulator - sends telemetry data to Go backend via TCP"
    )
    parser.add_argument(
        "--host", default="localhost", help="Go backend TCP host (default: localhost)"
    )
    parser.add_argument(
        "--port", type=int, default=61850, help="Go backend TCP port (default: 61850)"
    )
    parser.add_argument(
        "--interval",
        type=float,
        default=1.0,
        help="Send interval in seconds (default: 1.0)",
    )
    parser.add_argument(
        "--fault-mode",
        action="store_true",
        help="Increase overload probability to 20%% for alarm testing",
    )
    args = parser.parse_args()

    print(f"=== IEC 61850 Simulator ===")
    print(f"Target: {args.host}:{args.port}")
    print(f"Interval: {args.interval}s")
    print(f"Fault mode: {'ON' if args.fault_mode else 'OFF'}")
    print()

    try:
        asyncio.run(run_simulator(args.host, args.port, args.interval, args.fault_mode))
    except KeyboardInterrupt:
        print("\nSimulator stopped.")


if __name__ == "__main__":
    main()
