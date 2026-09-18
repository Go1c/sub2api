# Codex protocol reference

Selected runtime files from `maile456/codex-auto-sms-receiver`, commit
`1366a7156068d3172301c1c3487a23d3a2437458`, under its
`vendor/turb-gpt-free-register/` directory. Repository license retained in LICENSE.
The upstream directory itself derives from turb-gpt-free-register; sentinel/sdk.js
is an upstream-bundled service SDK, not original code authored by this project.

This snapshot runs only in the isolated login worker. Its application entry points,
registration, SMS, browser drivers, and remote CPA/sub2 integrations are not invoked.
Only the worker's explicit login steps run; HTTP destinations and mutating endpoints
are restricted there. Defaults in these reference config files are overridden.
Do not import this package into a shared application process. Its top-level
`config` namespace and synchronous network client are intentionally isolated
from the sub2api Go service and from the worker's HTTP request handler.

Source: https://github.com/maile456/codex-auto-sms-receiver/tree/1366a7156068d3172301c1c3487a23d3a2437458/vendor/turb-gpt-free-register

Local privacy patch: the Python/Node bridge passes cookies through stdin instead
of process arguments. Challenge files retain the upstream private temporary-file
handling and are deleted after use.
