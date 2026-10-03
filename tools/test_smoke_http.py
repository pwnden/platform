"""Check HTTP smoke coverage as problem catalogs and guidance change."""

import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

from smoke_http import check_catalog, check_guidance


class CatalogCoverageTests(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory(prefix="pwnden-catalog-coverage-")
        self.addCleanup(temporary.cleanup)
        self.work = Path(temporary.name)
        self.catalog = self.work / "config/pwnden/catalogs/fixture/catalog"
        self.slugs = ["note-vault", "rotor-lock", "new-exercise"]
        for slug in self.slugs:
            manifest = self.catalog / "challenges" / slug / "challenge.toml"
            manifest.parent.mkdir(parents=True)
            manifest.write_text(f'slug = "{slug}"\n')

    def test_additional_problem_is_included_without_registration(self):
        check_catalog(self.work, [{"slug": slug} for slug in reversed(self.slugs)])

    def test_missing_unexpected_and_duplicate_problems_fail(self):
        for slugs in (self.slugs[:2], [*self.slugs, "unexpected"], [*self.slugs, "new-exercise"]):
            with self.subTest(slugs=slugs), self.assertRaises(AssertionError):
                check_catalog(self.work, [{"slug": slug} for slug in slugs])

    def test_empty_installed_catalog_fails(self):
        for manifest in self.catalog.glob("challenges/*/challenge.toml"):
            manifest.unlink()
        with self.assertRaises(AssertionError):
            check_catalog(self.work, [])

    def test_catalog_cli_matches_installed_metadata(self):
        manifest = self.catalog / 'challenges/note-vault/challenge.toml'
        manifest.write_text('schema = 6\n[player]\ncli = ["nmap", "ncat"]\n')
        problems = [{"slug": slug} for slug in self.slugs]
        problems[0]['cli'] = ['nmap', 'ncat']
        check_catalog(self.work, problems)
        for names in [[], ['ncat', 'nmap'], ['curl'], None]:
            problems[0]['cli'] = names
            with self.subTest(names=names), self.assertRaises(AssertionError):
                check_catalog(self.work, problems)


class GuidanceCoverageTests(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory(prefix="pwnden-guidance-coverage-")
        self.addCleanup(temporary.cleanup)
        self.work = Path(temporary.name)
        self.directory = self.work / "config/pwnden/catalogs/fixture/catalog/challenges/example"
        self.directory.mkdir(parents=True)

    def fixture(self, count, *, walkthrough=True, schema=5, cli=()):
        paths = [f"notes/clue-{count-index}.md" for index in range(count)]
        declaration = f"schema = {schema}\n"
        if schema >= 2:
            declaration += "[content]\nhints = " + json.dumps(paths) + "\n"
            if walkthrough:
                declaration += 'walkthrough = "answer.md"\n'
        if schema >= 6:
            declaration += '[player]\ncli = ' + json.dumps(list(cli)) + '\n'
        (self.directory / "challenge.toml").write_text(declaration, encoding="utf-8")
        self.brief = {"hint_count": count, "walkthrough": walkthrough}
        if cli:
            self.brief['cli'] = list(cli)
        self.documents = {}
        for index, path in enumerate(paths, 1):
            target = self.directory / path
            target.parent.mkdir(exist_ok=True)
            text = f"힌트 {index}: declared order\n"
            target.write_text(text, encoding="utf-8")
            self.documents[f"hint-{index}"] = text
        if walkthrough:
            text = "Declared walkthrough\n"
            (self.directory / "answer.md").write_text(text, encoding="utf-8")
            self.documents["walkthrough"] = text

    def api(self, origin, token, method, path, data=None, expected=200):
        self.assertEqual((origin, method), ("http://fixture", "GET"))
        if token == "incorrect":
            self.assertEqual(expected, 401)
            return {"error": {"code": "unauthorized"}}
        self.assertEqual(token, "token")
        if path == "/problems/example":
            self.assertEqual(expected, 200)
            return self.brief
        id = path.removeprefix("/problems/example/guidance/")
        if id not in self.documents:
            self.assertEqual(expected, 404)
            return {"error": {"code": "not_found"}}
        self.assertEqual(expected, 200)
        return {"id": id, "content": self.documents[id]}

    def check(self):
        with patch("smoke_http.api", side_effect=self.api):
            check_guidance(self.work, "http://fixture", "token", "example")

    def test_declared_hint_counts_paths_order_and_optional_walkthrough(self):
        for count in (0, 1, 2, 4, 10):
            for walkthrough in (False, True):
                with self.subTest(count=count, walkthrough=walkthrough):
                    self.fixture(count, walkthrough=walkthrough)
                    self.check()

    def test_legacy_problem_has_no_guidance(self):
        self.fixture(0, walkthrough=False, schema=1)
        self.check()

    def test_detail_cli_matches_installed_metadata(self):
        self.fixture(1, schema=6, cli=['nmap', 'ncat'])
        self.check()
        for names in [[], ['ncat', 'nmap'], ['curl'], None]:
            self.brief['cli'] = names
            with self.subTest(names=names), self.assertRaises(AssertionError):
                self.check()

    def test_authored_line_endings_are_preserved(self):
        self.fixture(1)
        text = "Declared walkthrough\r\n"
        (self.directory / "answer.md").write_bytes(text.encode("utf-8"))
        self.documents["walkthrough"] = text
        self.check()

    def test_wrong_count_and_walkthrough_availability_fail(self):
        for field, value in (("hint_count", 3), ("hint_count", 1), ("walkthrough", False)):
            with self.subTest(field=field, value=value):
                self.fixture(2)
                self.brief[field] = value
                with self.assertRaises(AssertionError):
                    self.check()

    def test_swapped_changed_and_empty_guidance_fail(self):
        for corruption in ("swapped", "changed", "empty"):
            with self.subTest(corruption=corruption):
                self.fixture(2)
                if corruption == "swapped":
                    self.documents["hint-1"], self.documents["hint-2"] = self.documents["hint-2"], self.documents["hint-1"]
                else:
                    self.documents["walkthrough"] = "different document" if corruption == "changed" else " \n"
                with self.assertRaises(AssertionError):
                    self.check()


if __name__ == "__main__":
    unittest.main()
