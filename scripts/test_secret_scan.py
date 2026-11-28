"""Check narrow historical waivers and detect a new synthetic secret offline."""

import argparse
import json
from pathlib import Path
import re
import secrets
import subprocess
import tempfile


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--gitleaks-command", nargs=argparse.REMAINDER, default=["gitleaks"])
    args = parser.parse_args()
    repo = Path(__file__).resolve().parents[1]
    fingerprints = [
        line.strip()
        for line in (repo / ".gitleaksignore").read_text(encoding="utf-8").splitlines()
        if line.strip() and not line.startswith("#")
    ]
    assert len(fingerprints) == len(set(fingerprints)) == 12
    assert all(
        re.fullmatch(r"4f20c9896961fb3eab82851be97c324c82636350:[^:*?]+:[a-z0-9-]+:[0-9]+", value)
        for value in fingerprints
    ), "Only reviewed immutable fingerprints may be waived"

    # Keep the fixture outside .audit-tools, which is an intentionally excluded
    # local tools directory. This token is generated only for scanner testing;
    # it has never been registered with any provider or deployed as a credential.
    with tempfile.TemporaryDirectory(prefix="secret-scan-fixture-", dir=repo) as temporary:
        fixture = Path(temporary).resolve()
        assert fixture.parent == repo.resolve()
        (fixture / "sample.py").write_text('api_key = "' + secrets.token_urlsafe(48) + '"\n', encoding="utf-8")
        report = fixture / "redacted-report.json"
        result = subprocess.run(
            args.gitleaks_command + [
                "dir", str(fixture), "--config", str(repo / ".gitleaks.toml"),
                "--gitleaks-ignore-path", str(repo / ".gitleaksignore"),
                "--redact=100", "--no-banner", "--report-format=json", "--report-path=" + str(report),
            ],
            cwd=repo, capture_output=True, text=True,
        )
        assert result.returncode == 1, "The new synthetic secret must fail the scan"
        findings = json.loads(report.read_text(encoding="utf-8-sig"))
        assert any(row["RuleID"] == "generic-api-key" and row["File"].endswith("sample.py") for row in findings)
    print(json.dumps({"historical_waivers": 12, "new_synthetic_secret_detected": True}))


if __name__ == "__main__":
    main()
