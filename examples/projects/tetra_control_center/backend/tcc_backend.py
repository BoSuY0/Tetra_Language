from __future__ import annotations

import argparse
import json
import mimetypes
import os
import shutil
import subprocess
import time
from dataclasses import dataclass
from datetime import datetime, timezone
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from typing import Any
from urllib.parse import unquote, urlparse


APP_NAME = "Tetra Control Center"
ALLOWED_PROFILES = {
    "quiet": {
        "power_profile": "power-saver",
        "governor": "powersave",
        "epp": "power",
    },
    "balanced": {
        "power_profile": "balanced",
        "governor": "powersave",
        "epp": "balance_performance",
    },
    "performance": {
        "power_profile": "performance",
        "governor": "performance",
        "epp": "performance",
    },
    "custom": {
        "power_profile": None,
        "governor": None,
        "epp": None,
    },
}
CPU_GOVERNOR_VALUES = {"performance", "powersave", "schedutil", "ondemand", "conservative"}
CPU_EPP_VALUES = {"performance", "balance_performance", "balance_power", "power", "default"}


@dataclass
class CommandResult:
    returncode: int
    stdout: str
    stderr: str


class SystemCommands:
    def which(self, name: str) -> str | None:
        return shutil.which(name)

    def run(self, argv: list[str], timeout: int = 2) -> CommandResult:
        if not argv or any(not isinstance(part, str) for part in argv):
            return CommandResult(126, "", "invalid argv")
        try:
            completed = subprocess.run(
                argv,
                check=False,
                capture_output=True,
                text=True,
                timeout=timeout,
                shell=False,
            )
        except FileNotFoundError as exc:
            return CommandResult(127, "", str(exc))
        except subprocess.TimeoutExpired as exc:
            return CommandResult(124, exc.stdout or "", exc.stderr or "timeout")
        return CommandResult(completed.returncode, completed.stdout, completed.stderr)


@dataclass
class BackendContext:
    sys_root: Path
    proc_root: Path
    audit_log: Path
    allow_writes: bool = False
    commands: Any = None
    project_root: Path | None = None

    def __post_init__(self) -> None:
        if self.commands is None:
            self.commands = SystemCommands()
        self.sys_root = Path(self.sys_root)
        self.proc_root = Path(self.proc_root)
        self.audit_log = Path(self.audit_log)
        if self.project_root is not None:
            self.project_root = Path(self.project_root)


def now_iso() -> str:
    return datetime.now(timezone.utc).isoformat().replace("+00:00", "Z")


def read_text(path: Path) -> str | None:
    try:
        return path.read_text(errors="replace").strip()
    except OSError:
        return None


def read_first_existing(paths: list[Path]) -> str | None:
    for path in paths:
        value = read_text(path)
        if value not in (None, ""):
            return value
    return None


def parse_meminfo(path: Path) -> dict[str, int | None]:
    data: dict[str, int | None] = {"total_kb": None, "available_kb": None, "used_kb": None, "used_percent": None}
    raw = read_text(path)
    if not raw:
        return data
    values: dict[str, int] = {}
    for line in raw.splitlines():
        if ":" not in line:
            continue
        key, rest = line.split(":", 1)
        parts = rest.strip().split()
        if parts and parts[0].isdigit():
            values[key] = int(parts[0])
    total = values.get("MemTotal")
    available = values.get("MemAvailable")
    data["total_kb"] = total
    data["available_kb"] = available
    if total is not None and available is not None:
        used = max(total - available, 0)
        data["used_kb"] = used
        data["used_percent"] = round((used / total) * 100) if total else None
    return data


def parse_modules(path: Path) -> dict[str, bool]:
    interesting = [
        "tuxedo_keyboard",
        "tuxedo_io",
        "tuxedo_compatibility_check",
        "uniwill_laptop",
        "clevo_acpi",
        "nvidia",
        "nvidia_drm",
        "nvidia_modeset",
        "nouveau",
        "i915",
        "amdgpu",
    ]
    loaded = {name: False for name in interesting}
    raw = read_text(path) or ""
    for line in raw.splitlines():
        name = line.split(" ", 1)[0]
        if name in loaded:
            loaded[name] = True
    return loaded


def dmi_info(sys_root: Path) -> dict[str, str]:
    dmi_root = sys_root / "class/dmi/id"
    fields = [
        "sys_vendor",
        "product_name",
        "product_version",
        "board_vendor",
        "board_name",
        "board_version",
        "bios_vendor",
        "bios_version",
    ]
    return {field: read_text(dmi_root / field) or "unavailable" for field in fields}


def cpu_model(proc_root: Path) -> str:
    raw = read_text(proc_root / "cpuinfo") or ""
    for line in raw.splitlines():
        if line.lower().startswith("model name") and ":" in line:
            return line.split(":", 1)[1].strip()
    return "unavailable"


def unique(values: list[str]) -> list[str]:
    seen: set[str] = set()
    result: list[str] = []
    for value in values:
        if value and value not in seen:
            seen.add(value)
            result.append(value)
    return result


def cpufreq_dirs(sys_root: Path) -> list[Path]:
    roots = [
        sys_root / "devices/system/cpu",
        sys_root / "devices/system/cpu/cpufreq",
    ]
    paths: list[Path] = []
    for root in roots:
        paths.extend(root.glob("cpu*/cpufreq"))
        paths.extend(root.glob("policy*"))
    return sorted({path.resolve() for path in paths if path.exists()})


def cpu_info(sys_root: Path, proc_root: Path) -> dict[str, Any]:
    governors: list[str] = []
    epp: list[str] = []
    policies: list[dict[str, str]] = []
    for policy in cpufreq_dirs(sys_root):
        governor = read_text(policy / "scaling_governor")
        preference = read_text(policy / "energy_performance_preference")
        if governor:
            governors.append(governor)
        if preference:
            epp.append(preference)
        policies.append(
            {
                "path": str(policy),
                "governor": governor or "unavailable",
                "energy_performance_preference": preference or "unavailable",
            }
        )
    loadavg = read_text(proc_root / "loadavg") or "unavailable"
    return {
        "model": cpu_model(proc_root),
        "logical_cpus": os.cpu_count() or 0,
        "loadavg": loadavg,
        "governors": unique(governors),
        "epp": unique(epp),
        "policies": policies,
        "status": "supported" if policies else "unsupported",
        "reason": "cpufreq policy files discovered" if policies else "no cpufreq policy files found",
    }


def battery_info(sys_root: Path) -> list[dict[str, str]]:
    supplies = []
    for supply in sorted((sys_root / "class/power_supply").glob("*")):
        if not supply.exists():
            continue
        kind = read_text(supply / "type") or "Unknown"
        if kind != "Battery":
            continue
        item = {"name": supply.name, "type": kind}
        for field in [
            "status",
            "capacity",
            "charge_now",
            "charge_full",
            "energy_now",
            "energy_full",
            "power_now",
            "voltage_now",
            "manufacturer",
            "model_name",
        ]:
            value = read_text(supply / field)
            if value is not None:
                item[field] = value
        supplies.append(item)
    return supplies


def numeric_sensor_value(raw: str | None, scale: int = 1) -> float | None:
    if raw is None:
        return None
    try:
        value = int(raw)
    except ValueError:
        return None
    return round(value / scale, 1) if scale != 1 else float(value)


def hwmon_info(sys_root: Path) -> list[dict[str, Any]]:
    devices: list[dict[str, Any]] = []
    for device in sorted((sys_root / "class/hwmon").glob("hwmon*")):
        name = read_text(device / "name") or device.name
        temps = []
        fans = []
        pwm = []
        for path in sorted(device.glob("temp*_input")):
            temps.append({"name": path.name, "celsius": numeric_sensor_value(read_text(path), 1000)})
        for path in sorted(device.glob("fan*_input")):
            fans.append({"name": path.name, "rpm": numeric_sensor_value(read_text(path))})
        for path in sorted(device.glob("pwm*")):
            if path.name.endswith("_enable") or path.name.endswith("_mode"):
                continue
            pwm.append({"name": path.name, "present": True, "used_for_control": False})
        devices.append({"path": str(device), "name": name, "temperatures": temps, "fans": fans, "pwm": pwm})
    return devices


def power_profile_info(commands: Any) -> dict[str, Any]:
    if not commands.which("powerprofilesctl"):
        return {"status": "unsupported", "reason": "powerprofilesctl not found", "current": "unavailable", "available": []}
    current = commands.run(["powerprofilesctl", "get"], timeout=2)
    listing = commands.run(["powerprofilesctl", "list"], timeout=2)
    available = []
    for line in listing.stdout.splitlines():
        stripped = line.strip()
        if stripped.endswith(":") and not stripped.startswith(("CpuDriver", "PlatformDriver", "Degraded")):
            available.append(stripped.strip("* ").rstrip(":"))
    if current.returncode != 0:
        return {
            "status": "unsupported",
            "reason": (current.stderr or "powerprofilesctl get failed").strip(),
            "current": "unavailable",
            "available": unique(available),
        }
    return {
        "status": "supported",
        "reason": "powerprofilesctl is available",
        "current": current.stdout.strip() or "unavailable",
        "available": unique(available),
        "raw_list": listing.stdout.strip(),
    }


def nvidia_info(commands: Any) -> dict[str, Any]:
    if not commands.which("nvidia-smi"):
        return {"status": "unsupported", "reason": "nvidia-smi not found"}
    result = commands.run(
        [
            "nvidia-smi",
            "--query-gpu=name,driver_version,power.draw,temperature.gpu,utilization.gpu",
            "--format=csv,noheader,nounits",
        ],
        timeout=3,
    )
    if result.returncode != 0:
        return {"status": "unsupported", "reason": (result.stderr or result.stdout or "nvidia-smi failed").strip()}
    gpus = []
    for line in result.stdout.splitlines():
        parts = [part.strip() for part in line.split(",")]
        if len(parts) >= 5:
            gpus.append(
                {
                    "name": parts[0],
                    "driver_version": parts[1],
                    "power_draw_w": parts[2],
                    "temperature_c": parts[3],
                    "utilization_percent": parts[4],
                }
            )
    return {"status": "supported", "reason": "nvidia-smi query succeeded", "gpus": gpus}


def nbfc_info(commands: Any) -> dict[str, Any]:
    for name in ["nbfc", "nbfc-linux"]:
        path = commands.which(name)
        if not path:
            continue
        result = commands.run([name, "status", "-a"], timeout=3)
        if result.returncode == 0:
            return {"status": "supported", "reason": f"{name} status succeeded", "command": path, "raw_status": result.stdout.strip()}
        return {"status": "unsupported", "reason": f"{name} status failed: {(result.stderr or result.stdout).strip()}", "command": path}
    return {"status": "unsupported", "reason": "NBFC-Linux command not found"}


def tuxedo_info(sys_root: Path, dmi: dict[str, str], modules: dict[str, bool], commands: Any) -> dict[str, Any]:
    leds = [path.name for path in (sys_root / "class/leds").glob("*") if any(key in path.name.lower() for key in ["tuxedo", "uniwill", "clevo"])]
    platform = [
        str(path)
        for path in (sys_root / "devices/platform").glob("*")
        if any(key in path.name.lower() for key in ["tuxedo", "uniwill", "clevo"])
    ]
    vendor = dmi.get("sys_vendor", "")
    useful_interfaces = leds or platform
    command = commands.which("tuxedo-control-center")
    if "TUXEDO" in vendor.upper() and useful_interfaces:
        status = "supported"
        reason = "TUXEDO DMI and platform interfaces discovered"
    else:
        status = "unsupported"
        reason = "DMI is not TUXEDO and no useful tuxedo/uniwill/clevo LED, fan, or platform sysfs interface was discovered"
    return {
        "status": status,
        "reason": reason,
        "command": command or "not found",
        "modules": {key: value for key, value in modules.items() if key.startswith(("tuxedo", "uniwill", "clevo"))},
        "interfaces": {"leds": leds, "platform": platform},
    }


def fan_backend_info(hwmon: list[dict[str, Any]], nbfc: dict[str, Any]) -> dict[str, Any]:
    rpm_sensors = []
    pwm_entries = []
    for device in hwmon:
        for fan in device["fans"]:
            rpm_sensors.append({"device": device["name"], **fan})
        for pwm in device["pwm"]:
            pwm_entries.append({"device": device["name"], **pwm})
    if nbfc.get("status") == "supported":
        return {
            "status": "supported",
            "reason": "NBFC-Linux status works; writes still require explicit allow-writes and a validated NBFC config",
            "rpm_sensors": rpm_sensors,
            "raw_pwm": pwm_entries,
        }
    return {
        "status": "unsupported",
        "reason": "read-only RPM sensors only; no validated NBFC-Linux config/status and no allowlisted fan control backend",
        "rpm_sensors": rpm_sensors,
        "raw_pwm": pwm_entries,
    }


def driver_support(power: dict[str, Any], cpu: dict[str, Any], gpu: dict[str, Any], fans: dict[str, Any], tuxedo: dict[str, Any], nbfc: dict[str, Any]) -> list[dict[str, str]]:
    return [
        {"name": "power-profiles-daemon", "status": power["status"], "reason": power["reason"]},
        {"name": "CPU governor/EPP", "status": cpu["status"], "reason": cpu["reason"]},
        {"name": "NVIDIA", "status": gpu["nvidia"]["status"], "reason": gpu["nvidia"]["reason"]},
        {"name": "Fans", "status": fans["control"]["status"], "reason": fans["control"]["reason"]},
        {"name": "TUXEDO/DKMS/TCC", "status": tuxedo["status"], "reason": tuxedo["reason"]},
        {"name": "NBFC-Linux", "status": nbfc["status"], "reason": nbfc["reason"]},
    ]


def collect_snapshot(
    sys_root: Path = Path("/sys"),
    proc_root: Path = Path("/proc"),
    commands: Any | None = None,
    audit_log: Path | None = None,
) -> dict[str, Any]:
    commands = commands or SystemCommands()
    sys_root = Path(sys_root)
    proc_root = Path(proc_root)
    dmi = dmi_info(sys_root)
    modules = parse_modules(proc_root / "modules")
    cpu = cpu_info(sys_root, proc_root)
    hwmon = hwmon_info(sys_root)
    power = {"profile": power_profile_info(commands)}
    gpu = {"nvidia": nvidia_info(commands)}
    nbfc = nbfc_info(commands)
    tuxedo = tuxedo_info(sys_root, dmi, modules, commands)
    fans = {"hwmon": hwmon, "control": fan_backend_info(hwmon, nbfc)}
    memory = parse_meminfo(proc_root / "meminfo")
    battery = battery_info(sys_root)
    support = driver_support(power["profile"], cpu, gpu, fans, tuxedo, nbfc)
    return {
        "app": {"name": APP_NAME, "generated_at": now_iso(), "mode": "read-only" if audit_log else "snapshot"},
        "dashboard": {
            "cpu": cpu,
            "gpu": gpu,
            "memory": memory,
            "battery": battery,
            "sensors": hwmon,
            "power_profile": power["profile"],
            "driver_support": support,
        },
        "hardware": {"dmi": dmi, "kernel_modules": modules},
        "cpu": cpu,
        "memory": memory,
        "battery": battery,
        "sensors": {"hwmon": hwmon},
        "power": power,
        "gpu": gpu,
        "fans": fans,
        "diagnostics": {
            "dmi": dmi,
            "kernel_modules": modules,
            "sysfs_capabilities": {
                "cpufreq_policies": [policy["path"] for policy in cpu["policies"]],
                "hwmon_devices": [device["path"] for device in hwmon],
            },
            "tuxedo": tuxedo,
            "nbfc": nbfc,
            "support": support,
        },
        "profiles": {
            "available": list(ALLOWED_PROFILES.keys()),
            "current": power["profile"].get("current", "unavailable"),
            "dry_run_default": True,
        },
        "logs": read_audit_log(audit_log) if audit_log else [],
        "settings": {
            "allow_writes": False,
            "dry_run_default": True,
            "audit_log": str(audit_log) if audit_log else "disabled",
        },
    }


def read_audit_log(path: Path | None, limit: int = 80) -> list[dict[str, Any]]:
    if path is None or not path.exists():
        return []
    rows = []
    try:
        lines = path.read_text(errors="replace").splitlines()[-limit:]
    except OSError:
        return []
    for line in lines:
        try:
            rows.append(json.loads(line))
        except json.JSONDecodeError:
            rows.append({"raw": line})
    return rows


def write_audit(path: Path, entry: dict[str, Any]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    payload = {"timestamp": now_iso(), **entry}
    with path.open("a", encoding="utf-8") as handle:
        handle.write(json.dumps(payload, sort_keys=True) + "\n")


def planned_profile_operations(profile: str, sys_root: Path, commands: Any) -> list[dict[str, str]]:
    mapping = ALLOWED_PROFILES[profile]
    operations: list[dict[str, str]] = []
    if mapping["power_profile"] and commands.which("powerprofilesctl"):
        operations.append({"kind": "command", "target": "powerprofilesctl", "value": mapping["power_profile"]})
    for policy in cpufreq_dirs(sys_root):
        governor_path = policy / "scaling_governor"
        epp_path = policy / "energy_performance_preference"
        if mapping["governor"] and governor_path.exists():
            operations.append({"kind": "sysfs_write", "target": str(governor_path), "value": mapping["governor"]})
        if mapping["epp"] and epp_path.exists():
            operations.append({"kind": "sysfs_write", "target": str(epp_path), "value": mapping["epp"]})
    return operations


def allowed_sysfs_write(path: Path, sys_root: Path, value: str) -> bool:
    resolved = path.resolve()
    allowed_files = {"scaling_governor", "energy_performance_preference"}
    if path.name not in allowed_files:
        return False
    if path.name == "scaling_governor" and value not in CPU_GOVERNOR_VALUES:
        return False
    if path.name == "energy_performance_preference" and value not in CPU_EPP_VALUES:
        return False
    return any(resolved == (policy / path.name).resolve() for policy in cpufreq_dirs(sys_root))


def apply_profile(
    profile: str,
    dry_run: bool = True,
    allow_writes: bool = False,
    sys_root: Path = Path("/sys"),
    commands: Any | None = None,
    audit_log: Path = Path.home() / ".local/state/tetra-control-center/audit.jsonl",
) -> dict[str, Any]:
    commands = commands or SystemCommands()
    sys_root = Path(sys_root)
    normalized = str(profile or "").strip().lower()
    if normalized not in ALLOWED_PROFILES:
        result = {"status": "rejected", "reason": f"unsupported profile: {profile}", "planned_operations": []}
        write_audit(
            audit_log,
            {"action": "apply_profile", "profile": profile, "dry_run": dry_run, "allow_writes": allow_writes, "decision": "deny", "result": result},
        )
        return result
    operations = planned_profile_operations(normalized, sys_root, commands)
    if dry_run or not allow_writes:
        result = {
            "status": "dry-run",
            "reason": "writes disabled; planned allowlisted operations were not executed",
            "profile": normalized,
            "planned_operations": operations,
        }
        write_audit(
            audit_log,
            {
                "action": "apply_profile",
                "profile": normalized,
                "dry_run": dry_run,
                "allow_writes": allow_writes,
                "decision": "allow-dry-run",
                "operations": operations,
                "result": {"status": result["status"], "reason": result["reason"]},
            },
        )
        return result
    executed = []
    errors = []
    for operation in operations:
        if operation["kind"] == "command" and operation["target"] == "powerprofilesctl":
            completed = commands.run(["powerprofilesctl", "set", operation["value"]], timeout=5)
            executed.append(operation)
            if completed.returncode != 0:
                errors.append((completed.stderr or completed.stdout or "powerprofilesctl failed").strip())
            continue
        if operation["kind"] == "sysfs_write":
            target = Path(operation["target"])
            if not allowed_sysfs_write(target, sys_root, operation["value"]):
                errors.append(f"blocked non-allowlisted write: {target}")
                continue
            try:
                target.write_text(operation["value"] + "\n")
                executed.append(operation)
            except OSError as exc:
                errors.append(f"{target}: {exc}")
    status = "applied" if not errors else "partial"
    result = {
        "status": status,
        "reason": "; ".join(errors) if errors else "allowlisted operations applied",
        "profile": normalized,
        "planned_operations": operations,
        "executed_operations": executed,
    }
    write_audit(
        audit_log,
        {
            "action": "apply_profile",
            "profile": normalized,
            "dry_run": dry_run,
            "allow_writes": allow_writes,
            "decision": "allow-write",
            "operations": operations,
            "result": {"status": result["status"], "reason": result["reason"]},
        },
    )
    return result


def handle_api_request(method: str, path: str, body: Any, context: BackendContext) -> tuple[int, dict[str, Any]]:
    if method == "GET" and path == "/api/health":
        return 200, {"status": "ok", "app": APP_NAME, "allow_writes": context.allow_writes}
    if method == "GET" and path == "/api/snapshot":
        snapshot = collect_snapshot(context.sys_root, context.proc_root, context.commands, context.audit_log)
        snapshot["settings"]["allow_writes"] = context.allow_writes
        return 200, snapshot
    if method == "GET" and path == "/api/logs":
        return 200, {"logs": read_audit_log(context.audit_log)}
    if method == "POST" and path == "/api/profile":
        if not isinstance(body, dict):
            return 400, {"status": "rejected", "reason": "JSON object body required"}
        profile = str(body.get("profile", ""))
        dry_run = bool(body.get("dry_run", True))
        result = apply_profile(profile, dry_run=dry_run, allow_writes=context.allow_writes, sys_root=context.sys_root, commands=context.commands, audit_log=context.audit_log)
        return (200 if result["status"] != "rejected" else 400), result
    return 404, {"status": "not-found", "reason": "unknown allowlisted endpoint"}


class TCCRequestHandler(BaseHTTPRequestHandler):
    server_version = "TetraControlCenter/0.1"

    def context(self) -> BackendContext:
        return self.server.context  # type: ignore[attr-defined]

    def send_json(self, status: int, payload: dict[str, Any]) -> None:
        raw = json.dumps(payload, indent=2).encode("utf-8")
        self.send_response(status)
        self.send_header("Content-Type", "application/json; charset=utf-8")
        self.send_header("Content-Length", str(len(raw)))
        self.send_header("Cache-Control", "no-store")
        self.end_headers()
        self.wfile.write(raw)

    def read_json_body(self) -> Any:
        length = int(self.headers.get("Content-Length", "0") or "0")
        if length <= 0:
            return None
        raw = self.rfile.read(length)
        return json.loads(raw.decode("utf-8"))

    def do_GET(self) -> None:
        parsed = urlparse(self.path)
        if parsed.path.startswith("/api/"):
            status, payload = handle_api_request("GET", parsed.path, None, self.context())
            self.send_json(status, payload)
            return
        self.serve_static(parsed.path)

    def do_POST(self) -> None:
        parsed = urlparse(self.path)
        try:
            body = self.read_json_body()
        except json.JSONDecodeError:
            self.send_json(400, {"status": "rejected", "reason": "invalid JSON"})
            return
        status, payload = handle_api_request("POST", parsed.path, body, self.context())
        self.send_json(status, payload)

    def log_message(self, fmt: str, *args: Any) -> None:
        return

    def serve_static(self, raw_path: str) -> None:
        context = self.context()
        root = context.project_root or Path.cwd()
        if raw_path in ("", "/"):
            target = root / "web/index.html"
        else:
            relative = unquote(raw_path.lstrip("/"))
            target = root / relative
        try:
            resolved = target.resolve()
            root_resolved = root.resolve()
            resolved.relative_to(root_resolved)
            data = resolved.read_bytes()
        except (OSError, ValueError):
            self.send_error(404)
            return
        content_type = mimetypes.guess_type(str(resolved))[0] or "application/octet-stream"
        self.send_response(200)
        self.send_header("Content-Type", content_type)
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)


def default_audit_log() -> Path:
    return Path.home() / ".local/state/tetra-control-center/audit.jsonl"


def build_arg_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Tetra Control Center safe helper and web server")
    parser.add_argument("--host", default="127.0.0.1")
    parser.add_argument("--port", type=int, default=8765)
    parser.add_argument("--project-root", type=Path, default=Path(__file__).resolve().parents[1])
    parser.add_argument("--audit-log", type=Path, default=default_audit_log())
    parser.add_argument("--sys-root", type=Path, default=Path("/sys"))
    parser.add_argument("--proc-root", type=Path, default=Path("/proc"))
    parser.add_argument("--allow-writes", action="store_true", help="enable allowlisted writes; default is read-only")
    parser.add_argument("--snapshot", action="store_true", help="print one JSON snapshot and exit")
    return parser


def main(argv: list[str] | None = None) -> int:
    parser = build_arg_parser()
    args = parser.parse_args(argv)
    context = BackendContext(
        sys_root=args.sys_root,
        proc_root=args.proc_root,
        audit_log=args.audit_log,
        allow_writes=bool(args.allow_writes),
        project_root=args.project_root,
    )
    if args.snapshot:
        print(json.dumps(collect_snapshot(args.sys_root, args.proc_root, context.commands, args.audit_log), indent=2))
        return 0
    server = ThreadingHTTPServer((args.host, args.port), TCCRequestHandler)
    server.context = context  # type: ignore[attr-defined]
    print(f"{APP_NAME} serving http://{args.host}:{args.port}/")
    print(f"mode={'allow-writes' if context.allow_writes else 'read-only'} audit_log={context.audit_log}")
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        pass
    finally:
        server.server_close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
