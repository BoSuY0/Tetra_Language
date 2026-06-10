# Surface Animation Scheduler

Status: experimental production-scope evidence for the Surface Block-system
track.

Schema: `tetra.surface.animation-scheduler.v1`

Level: `production-animation-scheduler-v1`

Release scope: `surface-v1-linux-web`

Surface animation is Block-first. It is expressed through `lib.core.block`
`MotionSpec` values, deterministic motion frames, explicit invalidation, and
bounded lifecycle evidence. It is not a CSS animation runtime, a global
animation cascade, requestAnimationFrame parity, GPU compositor timing proof, or
permission to run unbounded hidden animation loops.

## Contract

Runtime reports that claim production-scoped Block motion evidence for
`examples/surface_block_motion.tetra` must include an `animation_scheduler`
object. The validator also checks the object whenever it is present in broader
Block-system reports.

Required fields:

- `schema`: `tetra.surface.animation-scheduler.v1`
- `level`: `production-animation-scheduler-v1`
- `source`: the same source path as the enclosing runtime report
- `release_scope`: `surface-v1-linux-web`
- `motion_quality_level`: `deterministic-block-motion-v1`
- `motion_clock`: `deterministic-test-clock-v1`
- `scheduler_policy`: `deterministic-motion-frame-scheduler-v1`
- `timeline_policy`: `stable-motion-timeline-v1`
- `invalidation_policy`: `motion-dirty-block-invalidation-v1`
- `lifecycle_policy`: `start-interpolate-settle-stop-v1`
- `reduced_motion_policy`: `instant-settle-no-schedule-v1`
- `frame_count`, `frame_budget`, `scheduled_frame_count`,
  `settled_frame_count`, and `reduced_motion_frame_count` derived from
  `motion_frames`
- `target_frame_interval_ms`, `max_frame_delta_ms`, and `jitter_budget_ms`
- `transition_properties` covering opacity, color, transform, translate, and
  scale
- deterministic timeline, frame timing, invalidation, lifecycle,
  reduced-motion, and visual-delta booleans
- `target_smoke` rows for the report target/runtime
- `negative_guards` for missing reduced motion, missing frame timing,
  unbounded frame schedules, unchanged visual frames, hidden animation loops,
  and CSS animation parity claims
- `nonclaims` for CSS animation runtime, global animation cascade,
  requestAnimationFrame parity, GPU compositor timing, and unbounded hidden
  animation loops

## Stdlib Surface

`lib.core.block` exposes the scheduler-facing helpers:

- `motion_frame_interval_ms()`
- `motion_frame_budget_default()`
- `motion_max_frame_delta_ms()`
- `motion_frame_timing_ok(previous_ms, current_ms)`
- `motion_lifecycle_complete_stops(motion, elapsed_ms)`
- `motion_reduced_stops_schedule(motion)`

These helpers are intentionally small. They describe the stable Block motion
boundary without promoting CSS, browser animation APIs, or target GPU timing as
Surface production requirements.

## Validation

`tools/validators/surface.ValidateReport` rejects production Block motion
reports when:

- `animation_scheduler` is absent for `examples/surface_block_motion.tetra`
- scheduler schema, level, source, release scope, policy, or clock mismatches
  the runtime report
- scheduler frame counts do not match `motion_frames`
- frame timing is missing or `max_frame_delta_ms` does not match observed frame
  deltas
- reduced motion does not instantly settle without scheduling
- target smoke does not cover the report target/runtime
- visual delta evidence is absent
- negative guards are missing
- required nonclaims are absent

The release evidence for this contract is generated under
`reports/surface-prod/P22-animation/`.
