# GitHub Plugin

This sample plugin validates the v2 plugin protocol. It is intentionally small:

- reads a JSON request from `stdin`
- emits JSON line events on `stdout`
- supports both `job` and `agent` modes
