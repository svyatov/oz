# Concepts

Shared domain vocabulary for this project — entities, named processes, and status concepts with
project-specific meaning. Seeded with core domain vocabulary, then accretes as ce-compound and
ce-compound-refresh process learnings; direct edits are fine. Glossary only, not a spec or catch-all.

## Wizard model

### Wizard
A configuration-defined interactive flow that walks a user through a series of options and assembles
a single shell command from their answers. A wizard declares its base command, optional version
detection, and an ordered list of options.

### Option
One step in a wizard — a single question whose answer contributes to the built command. An option
has a type (select, confirm, input, or multi-select) that determines how its answer renders: as a
flag with a value, a boolean flag, repeated flags, or a positional argument.

### Built command
The shell command a wizard produces from its answers, assembled as an ordered sequence of tagged
segments (the command words, positional arguments, and flags) rather than a raw string, so each
segment can be styled, executed, or compared independently.

## Versioning

### Version gating
Restricting which options — and which choices within an option — a wizard offers, based on the
tool's version expressed as a semver constraint. An option whose constraint excludes the active
version is dropped before the command is built, so a wizard only ever offers flags valid for that
version.

### Detected version
The tool version discovered at runtime by running the wizard's version command and parsing its
output. Drives version gating during a normal interactive run.

### Pinned version
A tool version supplied explicitly instead of detected — by a user pin or a test fixture — and used
for version gating in its place. Lets a flow target a specific version without the tool installed.

## Execution

### Hermetic build
Constructing a wizard's command with no live version detection and no shell execution for dynamic
choices, so identical inputs yield an identical command on any machine. Achieved by feeding a pinned
version into gating and taking dynamic-choice answers as literal values rather than running the
shell that would have generated those choices.
