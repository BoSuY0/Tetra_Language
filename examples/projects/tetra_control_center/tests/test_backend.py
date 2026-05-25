import json
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from backend import tcc_backend


class FakeCommands:
    def __init__(self):
        self.calls = []

    def which(self, name):
        return {
            "powerprofilesctl": "/usr/bin/powerprofilesctl",
            "nvidia-smi": "/usr/bin/nvidia-smi",
        }.get(name)

    def run(self, argv, timeout=2):
        self.calls.append(list(argv))
        if argv[:2] == ["powerprofilesctl", "get"]:
            return tcc_backend.CommandResult(0, "balanced\n", "")
        if argv[:2] == ["powerprofilesctl", "list"]:
            return tcc_backend.CommandResult(
                0,
                "* balanced:\n    CpuDriver:\tintel_pstate\n  performance:\n  power-saver:\n",
                "",
            )
        if argv and argv[0] == "nvidia-smi":
            return tcc_backend.CommandResult(
                0,
                "NVIDIA GeForce RTX 5050 Laptop GPU, 595.71.05, 12.2, 57, 6\n",
                "",
            )
        return tcc_backend.CommandResult(127, "", "not found")


class WritableFakeCommands(FakeCommands):
    def run(self, argv, timeout=2):
        self.calls.append(list(argv))
        if argv[:3] == ["powerprofilesctl", "set", "performance"]:
            return tcc_backend.CommandResult(0, "", "")
        return super().run(argv, timeout=timeout)


class BackendTests(unittest.TestCase):
    def test_collect_snapshot_reports_dmi_power_cpu_gpu_and_hwmon(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            sys_root = root / "sys"
            proc_root = root / "proc"
            (sys_root / "class/dmi/id").mkdir(parents=True)
            (sys_root / "class/hwmon/hwmon0").mkdir(parents=True)
            (sys_root / "devices/system/cpu/cpu0/cpufreq").mkdir(parents=True)
            (sys_root / "class/power_supply/BAT0").mkdir(parents=True)
            (proc_root).mkdir()
            (sys_root / "class/dmi/id/sys_vendor").write_text("DREAM MACHINES SP. Z O.O.\n")
            (sys_root / "class/dmi/id/product_name").write_text("V3xxSNP_SNN_SNM\n")
            (sys_root / "class/dmi/id/board_name").write_text("V3xxSNP_SNN_SNM\n")
            (sys_root / "class/hwmon/hwmon0/name").write_text("coretemp\n")
            (sys_root / "class/hwmon/hwmon0/temp1_input").write_text("61000\n")
            (sys_root / "class/hwmon/hwmon0/fan1_input").write_text("2300\n")
            (sys_root / "devices/system/cpu/cpu0/cpufreq/scaling_governor").write_text("powersave\n")
            (sys_root / "devices/system/cpu/cpu0/cpufreq/energy_performance_preference").write_text("balance_power\n")
            (sys_root / "class/power_supply/BAT0/type").write_text("Battery\n")
            (sys_root / "class/power_supply/BAT0/status").write_text("Discharging\n")
            (sys_root / "class/power_supply/BAT0/capacity").write_text("72\n")
            (proc_root / "meminfo").write_text("MemTotal: 1000000 kB\nMemAvailable: 250000 kB\n")
            (proc_root / "loadavg").write_text("0.10 0.20 0.30 1/100 42\n")
            (proc_root / "modules").write_text("tuxedo_compatibility_check 1 0 - Live 0x0\nnvidia 1 0 - Live 0x0\n")

            snapshot = tcc_backend.collect_snapshot(
                sys_root=sys_root,
                proc_root=proc_root,
                commands=FakeCommands(),
                audit_log=root / "audit.jsonl",
            )

        self.assertEqual(snapshot["hardware"]["dmi"]["sys_vendor"], "DREAM MACHINES SP. Z O.O.")
        self.assertEqual(snapshot["hardware"]["dmi"]["product_name"], "V3xxSNP_SNN_SNM")
        self.assertEqual(snapshot["power"]["profile"]["current"], "balanced")
        self.assertEqual(snapshot["cpu"]["governors"], ["powersave"])
        self.assertEqual(snapshot["cpu"]["epp"], ["balance_power"])
        self.assertEqual(snapshot["memory"]["total_kb"], 1000000)
        self.assertEqual(snapshot["battery"][0]["capacity"], "72")
        self.assertEqual(snapshot["gpu"]["nvidia"]["status"], "supported")
        self.assertEqual(snapshot["fans"]["control"]["status"], "unsupported")
        self.assertIn("read-only RPM sensors only", snapshot["fans"]["control"]["reason"])

    def test_apply_profile_rejects_unknown_profile_and_audits_denial(self):
        with tempfile.TemporaryDirectory() as tmp:
            audit_log = Path(tmp) / "audit.jsonl"
            result = tcc_backend.apply_profile(
                "turbo",
                dry_run=True,
                allow_writes=False,
                sys_root=Path(tmp) / "sys",
                commands=FakeCommands(),
                audit_log=audit_log,
            )

            self.assertEqual(result["status"], "rejected")
            self.assertIn("unsupported profile", result["reason"])
            audit = [json.loads(line) for line in audit_log.read_text().splitlines()]
            self.assertEqual(audit[-1]["decision"], "deny")
            self.assertEqual(audit[-1]["profile"], "turbo")

    def test_apply_profile_dry_run_does_not_write_sysfs_or_execute_set(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            policy = root / "sys/devices/system/cpu/cpu0/cpufreq"
            policy.mkdir(parents=True)
            governor = policy / "scaling_governor"
            epp = policy / "energy_performance_preference"
            governor.write_text("performance\n")
            epp.write_text("performance\n")
            commands = FakeCommands()

            result = tcc_backend.apply_profile(
                "quiet",
                dry_run=True,
                allow_writes=False,
                sys_root=root / "sys",
                commands=commands,
                audit_log=root / "audit.jsonl",
            )

            self.assertEqual(result["status"], "dry-run")
            self.assertEqual(governor.read_text(), "performance\n")
            self.assertEqual(epp.read_text(), "performance\n")
            self.assertNotIn(["powerprofilesctl", "set", "power-saver"], commands.calls)
            planned = {(op["kind"], op["target"], op["value"]) for op in result["planned_operations"]}
            self.assertIn(("command", "powerprofilesctl", "power-saver"), planned)
            self.assertIn(("sysfs_write", str(governor), "powersave"), planned)
            self.assertIn(("sysfs_write", str(epp), "power"), planned)

    def test_apply_profile_with_writes_only_updates_allowlisted_cpu_policy_files(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            policy = root / "sys/devices/system/cpu/cpu0/cpufreq"
            blocked = root / "sys/class/hwmon/hwmon0"
            policy.mkdir(parents=True)
            blocked.mkdir(parents=True)
            governor = policy / "scaling_governor"
            epp = policy / "energy_performance_preference"
            fan_pwm = blocked / "pwm1"
            governor.write_text("powersave\n")
            epp.write_text("balance_power\n")
            fan_pwm.write_text("80\n")
            commands = WritableFakeCommands()

            result = tcc_backend.apply_profile(
                "performance",
                dry_run=False,
                allow_writes=True,
                sys_root=root / "sys",
                commands=commands,
                audit_log=root / "audit.jsonl",
            )

            self.assertEqual(result["status"], "applied")
            self.assertEqual(governor.read_text(), "performance\n")
            self.assertEqual(epp.read_text(), "performance\n")
            self.assertEqual(fan_pwm.read_text(), "80\n")
            self.assertIn(["powerprofilesctl", "set", "performance"], commands.calls)
            self.assertFalse(tcc_backend.allowed_sysfs_write(fan_pwm, root / "sys", "255"))

    def test_api_handler_exposes_snapshot_and_profile_without_shell_executor(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / "sys/class/dmi/id").mkdir(parents=True)
            (root / "proc").mkdir()
            (root / "proc/meminfo").write_text("")
            (root / "proc/loadavg").write_text("")
            (root / "proc/modules").write_text("")
            context = tcc_backend.BackendContext(
                sys_root=root / "sys",
                proc_root=root / "proc",
                audit_log=root / "audit.jsonl",
                allow_writes=False,
                commands=FakeCommands(),
            )

            status, snapshot = tcc_backend.handle_api_request("GET", "/api/snapshot", None, context)
            self.assertEqual(status, 200)
            self.assertIn("dashboard", snapshot)

            status, result = tcc_backend.handle_api_request(
                "POST",
                "/api/profile",
                {"profile": "performance", "dry_run": True},
                context,
            )
            self.assertEqual(status, 200)
            self.assertEqual(result["status"], "dry-run")

            status, result = tcc_backend.handle_api_request(
                "POST",
                "/api/shell",
                {"cmd": "cat /etc/shadow"},
                context,
            )
            self.assertEqual(status, 404)
            self.assertNotIn("cat /etc/shadow", json.dumps(result))


if __name__ == "__main__":
    unittest.main()
