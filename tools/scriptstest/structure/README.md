# Structure Script Tests

This directory owns script-level tests for repository structure rules.

Keep structure tests here instead of adding more files to the flat
`tools/scriptstest/` package. This keeps the top-level script test directory on
the path toward the six-file budget while preserving release and CI coverage.
