#!/usr/bin/env python3
"""Render the offline, accessible project-progress dashboard from its JSON data."""
from html import escape
import json
from pathlib import Path
from m11_source_fingerprint import acceptance_matches, source_fingerprint

ROOT = Path(__file__).resolve().parents[1]
SOURCE = ROOT / "docs/project-progress.json"
TARGET = ROOT / "docs/PROJECT_PROGRESS.html"
data = json.loads(SOURCE.read_text(encoding="utf-8"))
# Show writer output separately from reviewed/published lessons. A draft counts
# only when all core metadata (including self-check), markdown, three specs,
# and three reference files are present and parseable.
manifest = json.loads((ROOT / "internal/content/manifest.json").read_text(encoding="utf-8"))
published = set(manifest["published_lessons"])
draft_ids = [f"middle-{n:02d}" for n in range(1, 41)] + [f"middleplus-{n:02d}" for n in range(1, 25)] + [f"senior-{n:02d}" for n in range(1, 25)]
draft_count = 0
for lesson_id in draft_ids:
    if lesson_id in published:
        continue
    try:
        meta = json.loads((ROOT / f"internal/content/data/{lesson_id}.json").read_text(encoding="utf-8"))
        body = ROOT / f"internal/content/data/{lesson_id}.md"
        exercises = meta["exercises"]
        required = ("title", "rank", "module", "estimated_minutes", "go_version",
                    "learning_outcomes", "glossary", "self_check", "markdown_file")
        if (meta["id"] != lesson_id or any(not meta.get(key) for key in required) or
                not body.is_file() or len(exercises) != 3):
            continue
        if all((ROOT / f"internal/grading/specs/{ex['id']}.json").is_file() and
               (ROOT / f"internal/content/solutions/{ex['id']}/main.go").is_file() for ex in exercises):
            draft_count += 1
    except (OSError, ValueError, KeyError, TypeError):
        continue
draft_track = next(item for item in data["workstreams"] if item.get("id") == "m11-drafts" or item["title"].startswith("M11 "))
draft_track["id"] = "m11-drafts"
draft_track["value"] = draft_count
acceptance_path = ROOT / data.get("m11_acceptance_report", "docs/validation/m11-current-tree-acceptance-20261016.txt")
current_fingerprint = source_fingerprint(ROOT)
acceptance_passed = False
acceptance_running = (
    data.get("m11_acceptance", {}).get("status") == "running"
    and data.get("m11_acceptance", {}).get("source_fingerprint") == current_fingerprint
)
if acceptance_path.is_file():
    evidence = acceptance_path.read_text(encoding="utf-8")
    acceptance_passed = acceptance_matches(evidence, current_fingerprint)
data["m11_acceptance"] = {"status": "passed" if acceptance_passed else ("running" if acceptance_running else "needs_recheck"),
                          "source_fingerprint": current_fingerprint}
draft_track["title"] = "M11 — подготовленные черновики"
draft_track["detail"] = (
    f"Подготовлены файлы {draft_count} из 88 глав. "
    + ("Полная автоматическая проверка текущих исходников прошла. "
       if acceptance_passed else ("Полная fingerprint-bound проверка исходников сейчас выполняется. "
                                  if acceptance_running else "Полную автоматическую проверку текущих исходников ещё нужно повторить. "))
    + data.get("m11_review_detail", "")
    + " Этот счётчик показывает подготовку файлов; проверка качества и публикация — отдельные условия."
)
data["phase_note"] = (
    f"Для Middle, Middle+ и Senior подготовлены {draft_count}/88 черновых глав. "
    + ("Автоматическая проверка текущих исходников прошла; главы ожидают педагогической приёмки."
       if acceptance_passed else ("Полный fingerprint-bound автоматический прогон текущего дерева выполняется."
                                  if acceptance_running else "Полный автоматический прогон нужно повторить после правок."))
    + data.get("phase_note_suffix", "")
)

assert {item["id"] for item in data["milestones"]} == {f"M{n}" for n in range(14)}
assert len(data["milestones"]) == 14
assert all(item["status"] in ("done", "in_progress", "not_started") for item in data["milestones"])
assert all(0 <= item["value"] <= item["max"] and item["max"] > 0 for item in data["workstreams"])

# Persist the derived file-presence count as part of the reproducible snapshot.
SOURCE.write_text(json.dumps(data, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")

def e(value):
    return escape(str(value), quote=True)

completed = sum(item["status"] == "done" for item in data["milestones"])
percent = round(completed * 100 / len(data["milestones"]))
rows = []
for item in data["workstreams"]:
    value, maximum = item["value"], item["max"]
    ratio = round(value * 100 / maximum)
    rows.append(f'''<article class="track"><div class="track-head"><h3>{e(item['title'])}</h3><strong>{value} / {maximum} {e(item['unit'])} <span>({ratio}%)</span></strong></div><progress value="{value}" max="{maximum}" aria-label="{e(item['title'])}">{ratio}%</progress><p>{e(item['detail'])}</p></article>''')
groups = {
    "done": "Готово",
    "in_progress": "Сейчас",
    "not_started": "Дальше",
}
sections = []
for key, title in groups.items():
    items = [item for item in data["milestones"] if item["status"] == key]
    cards = "".join(f'''<li class="milestone {key}"><span class="badge">{e(item['id'])}</span><span>{e(item['title'])}</span></li>''' for item in items)
    sections.append(f"<section class=\"milestone-group\"><h3>{title} <small>{len(items)}</small></h3><ol>{cards}</ol></section>")
f = data["forecast"]
html = f'''<!doctype html>
<html lang="ru">
<head>
<meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<meta name="color-scheme" content="light dark"><title>Прогресс Open Go Academy</title>
<style>
:root{{font-family:Inter,ui-sans-serif,system-ui,-apple-system,"Segoe UI",sans-serif;color:#172b3a;background:#f3f6f8;font-synthesis:none;text-rendering:optimizeLegibility}}
*{{box-sizing:border-box}}body{{margin:0;padding:32px 20px 56px}}main{{max-width:1000px;margin:auto}}
header{{padding:28px;background:linear-gradient(135deg,#fff,#e9f7f6);border:1px solid #d5e1e5;border-radius:18px}}
.kicker{{margin:0;color:#087e83;font-size:.8rem;font-weight:750;letter-spacing:.13em;text-transform:uppercase}}
h1{{font-size:clamp(2rem,5vw,3.4rem);line-height:1.08;margin:12px 0}}header p{{max-width:68ch;line-height:1.65;color:#475b69}}.updated{{font-size:.85rem}}
section{{margin-top:22px;padding:24px;background:#fff;border:1px solid #d8e2e7;border-radius:16px}}h2{{margin:0 0 16px;font-size:1.3rem}}h3{{margin:0;font-size:1rem}}
.overall{{display:grid;grid-template-columns:112px 1fr;gap:22px;align-items:center}}.number{{width:104px;height:104px;border-radius:50%;display:grid;place-items:center;background:conic-gradient(#0d9ca3 {percent}%,#dfe8eb 0);position:relative;font-size:1.6rem;font-weight:800}}.number::after{{content:"";position:absolute;inset:8px;background:#fff;border-radius:50%}}.number span{{z-index:1}}.note{{line-height:1.65;color:#425866;margin:.6rem 0 0}}
.track{{padding:16px 0;border-top:1px solid #e4eaed}}.track:first-of-type{{border:0}}.track-head{{display:flex;justify-content:space-between;gap:16px;align-items:baseline}}.track-head strong{{white-space:nowrap;font-variant-numeric:tabular-nums}}.track-head strong span{{color:#607481;font-weight:550}}progress{{width:100%;height:13px;margin:11px 0 2px;accent-color:#078b91}}.track p{{margin:.4rem 0 0;color:#526570;line-height:1.55;font-size:.94rem}}
.milestone-grid{{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:18px}}.milestone-group h3{{display:flex;justify-content:space-between;margin-bottom:12px}}.milestone-group h3 small{{color:#586c77}}ol{{list-style:none;padding:0;margin:0;display:grid;gap:8px}}.milestone{{display:grid;grid-template-columns:43px 1fr;gap:8px;align-items:start;padding:9px 10px;border-radius:9px;background:#f4f7f8;line-height:1.4;font-size:.91rem}}.milestone.done .badge{{color:#086a45}}.milestone.in_progress{{background:#e8f6f5}}.milestone.in_progress .badge{{color:#00666c}}.badge{{font-variant-numeric:tabular-nums;font-weight:800}}
.forecast{{border-left:4px solid #0d9ca3;padding-left:18px}}.forecast strong{{font-size:1.12rem}}.forecast p{{line-height:1.65;color:#425866}}footer{{padding:20px 6px;color:#5a6e79;font-size:.86rem;line-height:1.6}}
a{{color:#075e79}}:focus-visible{{outline:3px solid #bd5500;outline-offset:3px}}
@media(max-width:760px){{body{{padding:16px 12px 36px}}header,section{{padding:19px}}.milestone-grid{{grid-template-columns:1fr}}.track-head{{align-items:flex-start;flex-direction:column;gap:6px}}.overall{{grid-template-columns:82px 1fr;gap:14px}}.number{{width:78px;height:78px;font-size:1.2rem}}}}
@media(prefers-color-scheme:dark){{:root{{color:#e6edf1;background:#121b22}}header{{background:linear-gradient(135deg,#17232b,#12343a);border-color:#34454e}}section{{background:#1b2830;border-color:#364750}}.milestone{{background:#24343d}}.milestone.done .badge{{color:#80d6a3}}.milestone-group h3 small{{color:#bfccd2}}.milestone.in_progress{{background:#1d3d40}}.milestone.in_progress .badge{{color:#70d4d0}}.note,.track p,.forecast p,footer,.track-head strong span{{color:#bfccd2}}.number::after{{background:#1b2830}}.track{{border-color:#364750}}progress{{color:#3fc2c4}}}}
</style>
</head>
<body><main>
<header><p class="kicker">OPEN GO ACADEMY · ПАНЕЛЬ ПРОГРЕССА</p><h1>Движение к завершению проекта</h1><p>Снимок состояния разработки на {e(data['updated_at'])}. Показатели основаны на закрытых этапах и опубликованных критериях, а не на количестве написанных строк.</p><p class="updated">Текущий этап: <strong>{e(data['phase'])}</strong><br>{e(data['phase_note'])}</p></header>
<section><h2>Контрольные точки</h2><div class="overall"><div class="number" role="img" aria-label="Закрыто {completed} из {len(data['milestones'])} контрольных точек"><span>{percent}%</span></div><div><strong>Закрыто {completed} из {len(data['milestones'])} контрольных точек</strong><p class="note">Это доля этапов M0–M13. Этапы отличаются по размеру, поэтому показатель не равен доле затраченных усилий или готовности каждого курса.</p></div></div></section>
<section><h2>Основные части платформы</h2>{''.join(rows)}</section>
<section><h2>Этапы проекта</h2><div class="milestone-grid">{''.join(sections)}</div></section>
<section class="forecast"><h2>Примерный срок</h2><strong>{e(f['range'])}</strong><p><strong>{e(f.get('calendar_window',''))}</strong></p><p><strong>Основание:</strong> {e(f['basis'])}</p><p><strong>Уверенность:</strong> {e(f['confidence'])}</p><p>{e(f['constraints'])}</p></section>
<footer>Статус вручную не угадывается: после завершения каждого большого этапа данные обновляются и HTML пересобирается командой <code>python3 scripts/render-project-progress.py</code>. Полный запуск и проверки выполняются автоматически; ручное тестирование сайта владельцем не требуется. Эта страница показывает состояние локального репозитория и не обещает публикацию или работу после закрытия Codex.</footer>
</main></body></html>'''
TARGET.write_text(html, encoding="utf-8")
print(f"Generated {TARGET.relative_to(ROOT)}: {completed}/{len(data['milestones'])} gates ({percent}%), date {data['updated_at']}")
