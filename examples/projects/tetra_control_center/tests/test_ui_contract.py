import unittest
from pathlib import Path


PROJECT_ROOT = Path(__file__).resolve().parents[1]


class UIContractTests(unittest.TestCase):
    def test_tetra_source_declares_required_screens_profiles_and_events(self):
        source = (PROJECT_ROOT / "src/main.tetra").read_text()

        for screen in ["Dashboard", "Profiles", "Fans/Backends", "Diagnostics", "Logs", "Settings"]:
            self.assertIn(screen, source)

        for profile in ["Quiet", "Balanced", "Performance", "Custom"]:
            self.assertIn(profile, source)

        for event in ["openDashboard", "openProfiles", "openFans", "openDiagnostics", "openLogs", "openSettings"]:
            self.assertIn(f"event {event} ->", source)

        self.assertIn("state ControlCenterState:", source)
        self.assertIn("view ControlCenterView(state: ControlCenterState):", source)
        self.assertIn("bind driverSupportText", source)
        self.assertIn("bind fanSupportText", source)

    def test_web_host_derives_navigation_and_profiles_from_tetra_bundle(self):
        source = (PROJECT_ROOT / "web/app.mjs").read_text()

        self.assertIn("function tetraContract()", source)
        self.assertIn('field.name.startsWith("screen")', source)
        self.assertIn('field.name.startsWith("profile")', source)
        self.assertIn("tetra_control_center.ui.json", source)


if __name__ == "__main__":
    unittest.main()
