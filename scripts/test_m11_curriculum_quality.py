"""Regression checks for lesson-specific M11 pedagogy and self-study labs."""
import json
import re
import unittest
from collections import defaultdict
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
DATA = ROOT / "internal/content/data"
WORD = re.compile(r"[^\W_]+(?:[-'’][^\W_]+)*", re.UNICODE)


def lesson_ids():
    return ([f"middle-{n:02d}" for n in range(1, 41)]
            + [f"middleplus-{n:02d}" for n in range(1, 25)]
            + [f"senior-{n:02d}" for n in range(1, 25)])


def read_lesson(lesson_id):
    return json.loads((DATA / f"{lesson_id}.json").read_text(encoding="utf-8"))


def read_markdown(lesson_id):
    return (DATA / f"{lesson_id}.md").read_text(encoding="utf-8")


class M11CurriculumQualityTests(unittest.TestCase):
    def test_all_88_lessons_have_distinct_nonempty_outcomes_and_checks(self):
        outcomes = defaultdict(list)
        checks = defaultdict(list)
        self.assertEqual(len(lesson_ids()), 88)
        for lesson_id in lesson_ids():
            with self.subTest(lesson=lesson_id):
                meta = read_lesson(lesson_id)
                self.assertEqual(meta["id"], lesson_id)
                self.assertGreaterEqual(len(meta.get("learning_outcomes", [])), 3)
                self.assertGreaterEqual(len(meta.get("self_check", [])), 3)
                self.assertTrue(all(isinstance(x, str) and x.strip()
                                    for x in meta["learning_outcomes"]))
                for outcome in meta["learning_outcomes"]:
                    outcomes[outcome].append(lesson_id)
                for item in meta["self_check"]:
                    self.assertTrue(item.get("question", "").strip())
                    self.assertTrue(item.get("answer", "").strip())
                    checks[(item["question"].strip(), item["answer"].strip())].append(lesson_id)

        repeated_outcomes = {text: ids for text, ids in outcomes.items() if len(ids) > 1}
        repeated_checks = {pair: ids for pair, ids in checks.items() if len(ids) > 1}
        self.assertFalse(repeated_outcomes, f"copy-pasted outcomes: {repeated_outcomes}")
        self.assertFalse(repeated_checks, f"copy-pasted self-checks: {repeated_checks}")

    def test_advanced_lessons_meet_the_reading_length_contract(self):
        short_or_long = {}
        for lesson_id in lesson_ids():
            body = read_markdown(lesson_id)
            count = len(WORD.findall(body))
            if not 1800 <= count <= 2200:
                short_or_long[lesson_id] = count
        self.assertFalse(short_or_long, f"M11 lesson word counts outside 1800–2200: {short_or_long}")

    def test_middle_concurrency_bridges_and_system_labs_are_present(self):
        for number in range(1, 15):
            lesson_id = f"middle-{number:02d}"
            body = read_markdown(lesson_id)
            with self.subTest(lesson=lesson_id):
                self.assertIn("Практический мост", body, lesson_id)
                bridge = body.split("Практический мост", 1)[1].split("\n## ", 1)[0].lower()
                self.assertRegex(
                    bridge,
                    r"не (?:доказыва|подтвержда|проверя|запуска|созда|исполня|измеря)",
                    lesson_id,
                )
        for number in range(21, 38):
            lesson_id = f"middle-{number:02d}"
            body = read_markdown(lesson_id)
            lower = body.lower()
            with self.subTest(lesson=lesson_id):
                self.assertIn("воспроизводимая лабораторная", lower, lesson_id)
                self.assertTrue("ожидайте" in lower or "ожидаемое наблюдение" in lower,
                                f"{lesson_id}: lab needs an expected observation")
                self.assertTrue("очистка:" in lower or "внешних ресурсов не созда" in lower,
                                f"{lesson_id}: lab needs cleanup instructions")

    def test_middleplus_explanations_are_topic_specific(self):
        first_paragraphs = {}
        forbidden = "Producer (производитель) предлагает работу; admission control"
        for number in range(13, 24):
            lesson_id = f"middleplus-{number:02d}"
            body = read_markdown(lesson_id)
            with self.subTest(lesson=lesson_id):
                self.assertIn("## Зачем нужна эта граница", body)
                self.assertNotIn(forbidden, body)
                section = body.split("## Зачем нужна эта граница", 1)[1].split("\n## ", 1)[0]
                paragraph = next((p.strip() for p in section.split("\n\n") if p.strip()), "")
                self.assertGreater(len(paragraph), 100)
                first_paragraphs.setdefault(paragraph, []).append(lesson_id)
        repeated = {paragraph: ids for paragraph, ids in first_paragraphs.items() if len(ids) > 1}
        self.assertFalse(repeated, f"repeated Middle+ introductions: {repeated}")

    def test_targeted_first_use_glossaries_have_valid_translations(self):
        for number in (1, 2, 21, 24, 25, 30, 33):
            lesson_id = f"middle-{number:02d}"
            meta = read_lesson(lesson_id)
            terms = [item["term"] for item in meta.get("glossary", [])]
            with self.subTest(lesson=lesson_id):
                self.assertTrue(terms)
                self.assertEqual(len(terms), len(set(terms)))
                self.assertTrue(all(item.get("translation", "").strip()
                                    and item.get("definition", "").strip()
                                    for item in meta["glossary"]))
        middle_02 = read_lesson("middle-02")
        goroutine = next(item for item in middle_02["glossary"]
                         if item["term"] == "goroutine")
        self.assertIn("горутин", goroutine["translation"].lower())
        self.assertNotIn("горoutine", goroutine["translation"].lower())


if __name__ == "__main__":
    unittest.main()
