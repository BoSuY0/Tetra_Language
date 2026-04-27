# v0.1.2 TODO Internal Audit (2026-04-27)

Status: tracking note for release-readiness hygiene after the v0.1.2 gates.

## Why This Exists

`docs/plans/2026-04-27-tetra-v0_1-to-v1_0-full-todo.md` has all top-level
epics marked complete, but still contains many unchecked nested items. This
audit captures what is actually open so release notes and handoff docs do not
claim a fully closed inner checklist.

## Snapshot

Command used:

```sh
node - <<'JS'
const fs=require('fs');
const path='docs/plans/2026-04-27-tetra-v0_1-to-v1_0-full-todo.md';
const lines=fs.readFileSync(path,'utf8').split(/\n/);
let current='(before first epic)';
const open=[];
for(let i=0;i<lines.length;i++){
  const m=lines[i].match(/^- \[[ x]\] (\d+\. .*)/);
  if(m) current=m[1];
  if(/^\s*- \[ \]/.test(lines[i])) open.push({line:i+1,epic:current,text:lines[i].trim()});
}
const isProcess=(t)=>/Перевірити, чи всі згадані файли реально існують|Не закривати чекбокс без test\/evidence|Після зміни коду запускати focused tests|Після зміни docs запускати|Після зміни generated artifacts перевірити|Оновити цей TODO-file|Якщо задача виявилась post-v1/.test(t);
const isBlocker=(t)=>/Блокер\/залишок|Blocker:|still fails on this host|Next step:/.test(t);
const isDecision=(t)=>/post-v1|mandatory|scope|decision/i.test(t);
let process=0, blocker=0, decision=0, substantive=0;
for(const o of open){
  if(isProcess(o.text)) process++;
  else if(isBlocker(o.text)) blocker++;
  else if(isDecision(o.text)) decision++;
  else substantive++;
}
console.log(JSON.stringify({total:open.length,process,blocker,decision,substantive},null,2));
JS
```

Snapshot result:

- total open nested checkboxes: `389`
- process-template reminders: `196`
- blocker lines: `2`
- scope/decision lines: `8`
- substantive technical/doc items: `183`

## High-Count Areas

Top epics by nested open count in this snapshot:

- `37. Стабілізувати stdlib core modules.`: `15`
- `10. Завершити primitive і structural type contract.`: `14`
- `11. Стабілізувати type inference.`: `14`
- `12. Завершити optionals.`: `14`
- `13. Завершити typed errors.`: `14`
- `14. Стабілізувати enums і exhaustive match.`: `14`
- `15. Завершити generics MVP до v1 стабільності.`: `14`
- `16. Завершити protocols і conformance.`: `14`
- `17. Стабілізувати extensions.`: `14`
- `18. Довести ownership markers до v1 контракту.`: `14`
- `20. Стабілізувати unsafe і capabilities.`: `14`
- `21. Стабілізувати effects system.`: `14`

## Current Interpretation Rules

- Top-level `[x]` means the wave/epic was accepted in historical execution.
- Nested `[ ]` must be treated as real remaining work unless explicitly marked
  as historical/process-only and backed by evidence in the same branch state.
- Release and handoff docs should never state "no open checkboxes" for this
  TODO file unless nested counts are also zero.

## Next Cleanup Pass

1. Resolve one technical epic end-to-end per pass, then update nested status
   with exact command evidence.
2. Normalize process-template lines so they do not remain open indefinitely
   after accepted waves.
3. Keep this audit updated whenever release docs cite TODO closure.
