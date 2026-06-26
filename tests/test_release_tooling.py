import hashlib
import json
import subprocess
import tempfile
import unittest
from pathlib import Path


REPO = Path(__file__).resolve().parents[1]


class ReleaseToolingTest(unittest.TestCase):
    def test_openwrt_go_environment_patch_is_idempotent(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            makefile = Path(temporary) / "Makefile"
            makefile.write_text(
                "HOST_GO_VARS= \\\n"
                "\tGOENV=off \\\n"
                "\tCC=gcc\n"
                "PKG_GO_VARS= \\\n"
                "\tGOENV=off \\\n"
                "\tGOTOOLCHAIN=local \\\n"
                "\tCC=gcc\n",
                encoding="utf-8",
            )
            command = [REPO / "scripts/patch-openwrt-go-env.py", makefile]
            subprocess.run(command, check=True)
            first = makefile.read_text(encoding="utf-8")
            subprocess.run(command, check=True)
            second = makefile.read_text(encoding="utf-8")
            self.assertEqual(first, second)
            self.assertEqual(first.count("GOTOOLCHAIN=local"), 2)

    def test_assemble_release_validates_and_renames_artifacts(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            incoming = root / "incoming" / "build"
            incoming.mkdir(parents=True)
            files = {}
            for key, name in (
                ("daemon", "failsafe-dns-proxy.apk"),
                ("luci", "luci-app-failsafe-dns-proxy.apk"),
                ("i18n_ru", "luci-i18n-failsafe-dns-proxy-ru.apk"),
            ):
                path = incoming / name
                path.write_bytes(key.encode())
                files[key] = {
                    "file": name,
                    "sha256": hashlib.sha256(path.read_bytes()).hexdigest(),
                }
            metadata = {
                "openwrt_version": "25.12.4",
                "target": "mediatek",
                "subtarget": "mt7622",
                "pkgarch": "aarch64_cortex-a53",
                "package_manager": "apk",
                **files,
            }
            (incoming / "build-metadata.json").write_text(
                json.dumps(metadata), encoding="utf-8"
            )
            output = root / "release"
            subprocess.run(
                [
                    REPO / "scripts/assemble-release.py",
                    "--input",
                    root / "incoming",
                    "--output",
                    output,
                    "--tag",
                    "v0.2.0",
                    "--installer",
                    REPO / "scripts/install.sh",
                ],
                check=True,
            )
            manifest = json.loads((output / "manifest.json").read_text())
            self.assertEqual(manifest["release"], "v0.2.0")
            self.assertEqual(len(manifest["artifacts"]), 1)
            self.assertTrue((output / "install.sh").is_file())
            for package in ("daemon", "luci", "i18n_ru"):
                release_name = manifest["artifacts"][0][package]["file"]
                self.assertTrue(release_name.startswith("openwrt-25.12.4-"))
                self.assertTrue((output / release_name).is_file())

    def test_release_matrix_is_valid(self) -> None:
        result = subprocess.run(
            [REPO / "scripts/release-matrix.py"],
            check=True,
            capture_output=True,
            text=True,
        )
        matrix = json.loads(result.stdout)
        self.assertEqual(len(matrix["include"]), 2)
        targets = {
            (item["openwrt_version"], item["target"], item["subtarget"])
            for item in matrix["include"]
        }
        self.assertEqual(
            targets,
            {
                ("24.10.7", "mediatek", "mt7622"),
                ("25.12.4", "mediatek", "mt7622"),
            },
        )
        managers = {item["package_manager"] for item in matrix["include"]}
        self.assertEqual(managers, {"opkg", "apk"})


if __name__ == "__main__":
    unittest.main()
