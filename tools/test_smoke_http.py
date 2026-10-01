"""Check HTTP smoke catalog coverage as authored problems are added."""

from pathlib import Path
import tempfile
import unittest

from smoke_http import check_catalog


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


if __name__ == "__main__":
    unittest.main()
