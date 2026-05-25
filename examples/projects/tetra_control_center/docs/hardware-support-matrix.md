# Hardware Support Matrix

Observed on May 23, 2026 local time on:

- Vendor: `DREAM MACHINES SP. Z O.O.`
- Product: `V3xxSNP_SNN_SNM`
- Board: `V3xxSNP_SNN_SNM`
- BIOS: `INSYDE Corp. 1.07.04TDES`

## Evidence Commands

```sh
for f in sys_vendor product_name board_vendor board_name bios_vendor bios_version; do
  cat "/sys/class/dmi/id/$f"
done

powerprofilesctl get
powerprofilesctl list
nvidia-smi --query-gpu=name,driver_version,power.draw,power.limit,temperature.gpu,utilization.gpu --format=csv,noheader,nounits
sensors
find /sys/class/hwmon -maxdepth 1 -mindepth 1 -print
find /sys/class/leds -maxdepth 1 -mindepth 1 -printf '%f\n'
cut -d' ' -f1 /proc/modules | rg '^(tuxedo|uniwill|clevo|nvidia|nouveau|i915|amdgpu)'
command -v nbfc || true
command -v nbfc-linux || true
```

## Matrix

| Area | Status | Evidence | Notes |
| --- | --- | --- | --- |
| DMI identification | Supported | DMI reports Dream Machines `V3xxSNP_SNN_SNM` | Used for diagnostics only. |
| `powerprofilesctl` | Supported | `powerprofilesctl get` returned `performance`; list showed `performance`, `balanced`, `power-saver` | Safe adapter supports dry-run and allowlisted `set`. |
| CPU governor/EPP | Supported | cpufreq policies expose `scaling_governor=performance` and `energy_performance_preference=performance` | Writes require `--allow-writes`; dry-run is default. |
| `hwmon` sensors | Supported read-only | `coretemp`, `BAT0`, `nvme`, `iwlwifi_1`, `acpitz`, `spd5118`, and `acpi_fan` devices found | Used for temperatures and fan RPM only. |
| Fan RPM | Supported read-only | `sensors` showed multiple `acpi_fan` RPM readings | This is not fan control. |
| Fan control | Unsupported | NBFC-Linux command not found; no validated control backend | No EC/fan register writes are implemented. |
| RGB/keyboard backlight | Unsupported | `/sys/class/leds` did not expose TUXEDO/uniwill/clevo keyboard LED interfaces | No RGB support claimed. |
| NVIDIA info | Supported read-only | `nvidia-smi` returned `NVIDIA GeForce RTX 5050 Laptop GPU`, driver `595.71.05` | Info only; no unsafe GPU control path. |
| TUXEDO/DKMS/TCC | Unsupported | DMI is Dream Machines; only `tuxedo_compatibility_check` module observed; no useful TUXEDO/uniwill platform interfaces | TUXEDO Control Center is not used as the control backend. |
| NBFC-Linux | Unsupported | `nbfc` and `nbfc-linux` commands were not found | Adapter remains status-only until config/status works. |

## Current Support Statement

The app supports evidence-backed read-only diagnostics plus allowlisted power
profile and CPU policy operations. It does not support fan control, RGB control,
EC writes, or TUXEDO-specific laptop control on this hardware.
