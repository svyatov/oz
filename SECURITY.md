# Security policy

## Supported versions

Only the latest release receives security fixes. oz is at `0.x`, so a fix ships in the next minor
release rather than as a patch to an older line.

| Version | Supported |
|---------|-----------|
| 0.2.x   | Yes       |
| < 0.2   | No        |

## Reporting a vulnerability

Report privately through GitHub's security advisory form:

<https://github.com/svyatov/oz/security/advisories/new>

The form is open to anyone with a GitHub account and the report stays private until an advisory is
published. Do not open a public issue for a suspected vulnerability.

You will get an initial response within 14 days. If you have not heard back by then, escalate by
opening a public issue that says a private report is outstanding, without describing the
vulnerability itself.

Include what you have: the oz version (`oz --version`), the wizard config that triggers the problem,
the command you ran, what happened, and what you expected instead. A reproduction case matters more
than a severity rating.

## What counts

oz builds a shell command from a YAML wizard config and runs it, so a wizard that runs a command you
told it to run is working as designed. The interesting reports are the ones where oz does something
the config did not ask for:

- A wizard config or registry response that makes oz execute something the config does not declare.
- A password field value reaching a place the README says it never reaches: the persisted last-used
  state, a preset, a pin, or oz's own printed output.
- A path in `oz add` or `oz update` that writes outside the config directory.
- A dependency vulnerability that oz reaches at run time.

Running an untrusted wizard config is equivalent to running an untrusted script. Review a config
before you run it, the same way you would review a shell script.

## Disclosure

Fixes are released before details are published. Once a fix ships, the advisory goes public and the
changelog entry links to it. Credit goes to the reporter unless they ask otherwise.
