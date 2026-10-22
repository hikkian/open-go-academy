#!/usr/bin/env python3
"""Light checks for lessons written by docs/LESSON_STYLE_GUIDE.md: JSON fields, task cards, links, callout labels,
expected-output blocks against the grader, and every Go example run one by one with the pinned toolchain.
    python3 scripts/check_lesson_template.py zero-01 zero-02
Not a replacement for make test (which also runs the examples through the content package)."""
import json, re, subprocess, sys, os, tempfile
ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
GO = os.path.join(ROOT, ".tools/go/bin/go")
WORD = re.compile(r"[A-Za-zА-Яа-яЁё0-9_]+")
problems = []
def slug(title):
    out=[]; space=False
    for ch in title.strip().lower():
        if ch in " \t": space = bool(out); continue
        if ch.isalnum() or ch in "-_" or ('Ѐ'<=ch<='ӿ'):
            if space: out.append('-'); space=False
            out.append(ch)
    return "".join(out)
def examples(md):
    groups=[]; active=None; code=[]
    for line in md.split("\n"):
        if active is None:
            f=line.strip().split()
            if not f or f[0]!="```go": continue
            mode="run"
            for a in f[1:]:
                if a.startswith("mode="): mode=a[5:]
                elif not (a.startswith("group=") or a.startswith("file=")): problems.append(f"unknown go attribute {a}")
            active=mode; code=[]; continue
        if line.strip()=="```": groups.append((active,"\n".join(code)+"\n")); active=None; continue
        code.append(line)
    if active is not None: problems.append("unterminated go block")
    return groups
def links(md, lesson_ids, exercise_ids):
    body=re.sub(r"```.*?```", "", md, flags=re.S)
    for m in re.finditer(r"\]\(([^)]+)\)", body):
        t=m.group(1)
        if "://" in t: continue
        if t.startswith("lesson:"):
            if t[7:] not in lesson_ids: problems.append(f"missing lesson link {t}")
        elif t.startswith("exercise:"):
            if t[9:] not in exercise_ids: problems.append(f"missing exercise link {t}")
        elif t.startswith("#"):
            heads=[slug(l.lstrip('#')) for l in md.split("\n") if l.startswith("#")]
            if t[1:] not in heads: problems.append(f"missing heading link {t}")
        else: problems.append(f"unsupported link {t}")
manifest=json.load(open(f"{ROOT}/internal/content/manifest.json"))
lesson_ids=set(manifest["published_lessons"])
all_ex=set()
for lid in lesson_ids:
    for e in json.load(open(f"{ROOT}/internal/content/data/{lid}.json")).get("exercises",[]): all_ex.add(e["id"])
for lid in sys.argv[1:]:
    d=json.load(open(f"{ROOT}/internal/content/data/{lid}.json")); md=open(f"{ROOT}/internal/content/data/{lid}.md",encoding="utf-8").read()
    print(f"== {lid}: {d['title']} | words {len(WORD.findall(md))} | exercises {len(d['exercises'])} | self_check {len(d['self_check'])}")
    if not md.startswith("# "+d["title"]+"\n"): problems.append(f"{lid}: first line is not the title")
    for k in ("rank","module","estimated_minutes","go_version","learning_outcomes","self_check","glossary"):
        if not d.get(k): problems.append(f"{lid}: empty {k}")
    for e in d["exercises"]:
        for k in ("id","title","prompt","starter_file","starter_code","criteria"):
            if not e.get(k): problems.append(f"{lid}/{e['id']}: empty {k}")
        if len(e["prompt"])>90: problems.append(f"{lid}/{e['id']}: prompt {len(e['prompt'])} chars > 90")
    cards=re.findall(r"^### Задание (\d+) · (.+)$", md, flags=re.M)
    want=[(str(i+1), e["title"]) for i,e in enumerate(d["exercises"])]
    if cards!=want: problems.append(f"{lid}: cards {cards} != exercises {want}")
    # expected-output block right after each card's «Ожидаемый вывод» line
    for i,e in enumerate(d["exercises"]):
        sec=md.split(f"### Задание {i+1} · {e['title']}",1)[1].split("\n### ",1)[0]
        m=re.search(r"\*\*Ожидаемый вывод\*\*.*?\n```text\n(.*?)```", sec, flags=re.S)
        if not m: problems.append(f"{lid}/{e['id']}: no expected-output block"); continue
        block=m.group(1)
        # The block must match the program's real output byte for byte: that is the text the learner has to print.
        # public_output is that output (the Go encoder of the grader writes it, e.g. [1,2,3] with no spaces).
        # The grader drops one final newline before it compares lines (grading.go: TrimSuffix), and public_output may
        # lack it; a fenced block always ends with a newline. So the block matches public_output or public_output + "\n".
        exp=e.get("public_output")
        if not exp: problems.append(f"{lid}/{e['id']}: no public_output to compare the expected-output block with"); continue
        if block not in (exp, exp+"\n"): problems.append(f"{lid}/{e['id']}: expected-output block differs from public_output: {block!r} vs {exp!r}")
        for lab in ("**Цель.**","**Шаги.**","**Как проверить себя.**","**Если не принято.**","> **Подсказка.**"):
            if lab not in sec: problems.append(f"{lid}/{e['id']}: card lacks {lab}")
    for lab in re.findall(r"^> \*\*([^*]+)\*\*", md, flags=re.M):
        if lab not in ("Важно.","Подсказка.","Типичная ошибка.","Зачем."): problems.append(f"{lid}: unknown callout label {lab}")
    if re.search(r"^\|", md, flags=re.M): problems.append(f"{lid}: markdown table (not rendered)")
    # <nil> is Go's printed output (fmt prints a nil error as <nil>), not markup
    if re.search(r"<[a-z]+[ >]", md.replace("<nil>", "")): problems.append(f"{lid}: raw HTML")
    links(md, lesson_ids, all_ex)
    ex=examples(md)
    if not ex: problems.append(f"{lid}: no go example")
    for n,(mode,code) in enumerate(ex,1):
        with tempfile.TemporaryDirectory() as tmp:
            open(f"{tmp}/main.go","w").write(code)
            cmd=["nice","-n","19",GO,"run" if mode=="run" else "vet","main.go"]
            r=subprocess.run(cmd,cwd=tmp,capture_output=True,text=True,timeout=120)
            status="ok" if r.returncode==0 else "FAIL"
            print(f"   example {n} ({mode}): {status}", (r.stdout.strip().replace(chr(10),' | ')[:90] if r.returncode==0 else r.stderr[:300]))
            if r.returncode!=0: problems.append(f"{lid}: go example {n} failed")
print("\nPROBLEMS:" if problems else "\nALL CHECKS PASSED"); [print(" -",p) for p in problems]
sys.exit(1 if problems else 0)
