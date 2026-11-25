"""Exercise the compiled CLI, real WASM components and native driver boundaries.

Usage: python3 agent-os/tests/e2e.py /absolute/path/to/arcloom-os
Build/pack both example components first (examples/demo.sh does this).
"""
import copy
import json
import pathlib
import subprocess
import sys
import tempfile
import time
import unittest

if len(sys.argv) < 2:
    raise SystemExit(__doc__)
BINARY = pathlib.Path(sys.argv.pop(1)).resolve(strict=True)
EXAMPLES = pathlib.Path(__file__).resolve().parents[1] / "examples"


class AgentOsCliTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory(prefix="arcloom-e2e-")
        self.addCleanup(self.temporary.cleanup)
        self.directory = pathlib.Path(self.temporary.name)
        self.state = self.directory / "state"
        self.manifest = json.loads((EXAMPLES / "feedback-loop.json").read_text())
        self.manifest["component"] = str(
            (EXAMPLES / self.manifest["component"]).resolve(strict=True)
        )
        for driver in self.manifest["drivers"]:
            driver["command"]["cwd"] = str(EXAMPLES)
        self.config = self.directory / "manifest.json"
        self.save()

    def save(self):
        self.config.write_text(json.dumps(self.manifest))

    def cli(self, *args, expected=0):
        result = subprocess.run(
            [str(BINARY), *map(str, args)],
            text=True,
            capture_output=True,
            timeout=30,
        )
        self.assertEqual(
            result.returncode,
            expected,
            f"command {args}: stdout={result.stdout!r}; stderr={result.stderr!r}",
        )
        return result

    def run_task(self, expected=0, run_id="example"):
        self.save()
        result = self.cli("run", self.config, self.state, run_id, expected=expected)
        return json.loads(result.stdout)

    def inspect(self, run_id="example"):
        return json.loads(self.cli("inspect", self.state, run_id).stdout)

    def set_execution(self, source, timeout_ms=3000):
        self.manifest["drivers"][0]["command"]["argv"] = [
            "/usr/bin/python3", "-c", source
        ]
        self.manifest["drivers"][0]["command"]["timeout_ms"] = timeout_ms

    def test_feedback_loop_and_idempotent_completed_resume(self):
        first = self.run_task()
        self.assertEqual(first["status"], "completed")
        self.assertEqual(first["actions"], 2)
        self.assertEqual(first["state"]["result"], {"text": "HELLO ARCLOOM"})
        before = self.inspect()
        self.assertEqual(self.run_task(), first)
        self.assertEqual(self.inspect(), before)

    def test_invalid_application_result_cannot_dispatch(self):
        del self.manifest["intent"]["criteria"]
        self.save()
        self.cli("run", self.config, self.state, "example", expected=1)
        result = self.inspect()["run"]
        self.assertEqual(result["actions"], 0)
        self.assertIsNone(result["state"])

    def test_observe_only_has_no_mandatory_driver_sequence(self):
        self.manifest = json.loads((EXAMPLES / "observe-only.json").read_text())
        self.manifest["component"] = str(
            (EXAMPLES / self.manifest["component"]).resolve(strict=True)
        )
        result = self.run_task()
        self.assertEqual(result["status"], "completed")
        self.assertEqual(result["actions"], 0)
        self.assertEqual(result["state"]["observed_intent"], self.manifest["intent"])

    def test_replace_execution_driver_without_recompiling_application(self):
        selected = "result from a substituted provider"
        self.manifest["intent"]["criteria"]["expected"] = selected
        self.manifest["drivers"][0]["command"]["env"] = {"RESULT": selected}
        self.set_execution(
            "import json, os; print(json.dumps({'text': os.environ['RESULT']}))"
        )
        result = self.run_task()
        self.assertEqual(result["status"], "completed")
        self.assertEqual(result["actions"], 2)
        self.assertEqual(result["state"]["result"]["text"], selected)

    def test_missing_grant_denies_before_external_action(self):
        marker = self.directory / "must-not-exist"
        self.set_execution(f"from pathlib import Path; Path({str(marker)!r}).touch()")
        self.manifest["grants"] = []
        result = self.run_task(expected=2)
        self.assertEqual(result["status"], "denied")
        self.assertEqual(result["actions"], 0)
        self.assertFalse(marker.exists())

    def test_rejected_feedback_retries_until_configured_budget(self):
        self.manifest["intent"]["criteria"]["expected"] = "unreachable result"
        result = self.run_task(expected=2)
        self.assertEqual(result["status"], "budget_hit")
        self.assertEqual(result["actions"], 4)
        signals = self.inspect()["signals"]
        feedback = [item for item in signals if item["kind"] == "feedback"]
        self.assertEqual(len(feedback), 2)
        self.assertTrue(all(item["payload"]["accepted"] is False for item in feedback))

    def test_changed_intent_and_component_cannot_resume_old_binding(self):
        self.run_task()
        before = self.inspect()
        original = copy.deepcopy(self.manifest)
        self.manifest["intent"]["goal"] = "a different goal"
        self.save()
        rejected = self.cli("run", self.config, self.state, "example", expected=1)
        self.assertIn("binding changed", rejected.stderr)
        self.assertEqual(self.inspect(), before)
        self.manifest = original
        self.manifest["component"] = str(
            EXAMPLES / "generated" / "observe-only.component.wasm"
        )
        self.save()
        rejected = self.cli("run", self.config, self.state, "example", expected=1)
        self.assertIn("binding changed", rejected.stderr)
        self.assertEqual(self.inspect(), before)

    def test_wasm_fuel_exhaustion_cannot_dispatch_external_action(self):
        marker = self.directory / "must-not-exist"
        self.set_execution(f"from pathlib import Path; Path({str(marker)!r}).touch()")
        self.manifest["limits"]["fuel"] = 1
        self.save()
        rejected = self.cli("run", self.config, self.state, "example", expected=1)
        self.assertTrue(rejected.stderr)
        self.assertFalse(marker.exists())
        self.assertEqual(self.inspect()["run"]["actions"], 0)

    def test_timeout_is_uncertain_and_is_not_replayed_on_resume(self):
        attempts = self.directory / "attempts"
        self.set_execution(
            f"import time; open({str(attempts)!r}, 'a').write('attempt\\n'); time.sleep(3)",
            timeout_ms=100,
        )
        first = self.run_task(expected=2)
        self.assertEqual(first["status"], "uncertain")
        self.assertEqual(first["actions"], 1)
        self.assertEqual(attempts.read_text(), "attempt\n")
        before = self.inspect()
        self.assertEqual(self.run_task(expected=2), first)
        self.assertEqual(self.inspect(), before)
        self.assertEqual(attempts.read_text(), "attempt\n")

    def test_concurrent_host_cannot_enter_locked_state_directory(self):
        started = self.directory / "started"
        release = self.directory / "release"
        self.set_execution(
            "import json, time; from pathlib import Path\n"
            f"Path({str(started)!r}).touch()\n"
            f"while not Path({str(release)!r}).exists(): time.sleep(0.01)\n"
            "print(json.dumps({'text': 'HELLO ARCLOOM'}))\n",
            timeout_ms=10000,
        )
        self.save()
        first = subprocess.Popen(
            [str(BINARY), "run", str(self.config), str(self.state), "example"],
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
        )
        try:
            deadline = time.monotonic() + 10
            while not started.exists() and time.monotonic() < deadline:
                if first.poll() is not None:
                    self.fail(f"first host ended before marker: {first.communicate()}")
                time.sleep(0.01)
            self.assertTrue(started.exists(), "first driver never started")
            rejected = self.cli("run", self.config, self.state, "second", expected=1)
            self.assertIn("already in use", rejected.stderr)
        finally:
            release.touch()
            try:
                stdout, stderr = first.communicate(timeout=15)
            except subprocess.TimeoutExpired:
                first.kill()
                stdout, stderr = first.communicate()
                self.fail(f"first host failed to stop: {stdout!r}, {stderr!r}")
        self.assertEqual(first.returncode, 0, stderr)
        self.assertEqual(json.loads(stdout)["status"], "completed")


if __name__ == "__main__":
    unittest.main(verbosity=2)
