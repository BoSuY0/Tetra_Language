# Dogfood Actor/Task

Deterministic actor/task project for release smoke. The task side returns a
stable score, the actor side echoes a tagged value, and the main function checks
both paths without unbounded scheduling.

Expected exit code: `0`.

Typed actor messages are limited to the current `actor.msg` value/tag envelope.
Cancellation is a documented non-goal for the v1 cooperative task MVP.

