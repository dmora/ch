# Security Policy

## Scope

`ch` is a read-only CLI tool that accesses Claude Code's local conversation cache.
It does not:
- Make network requests
- Modify any files
- Execute external commands (except `claude --resume`)

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| latest  | :white_check_mark: |

## Reporting a Vulnerability

If you discover a security vulnerability, please report it via
[GitHub Security Advisories](https://github.com/dmora/ch/security/advisories/new).

Please include:
- Description of the vulnerability
- Steps to reproduce
- Potential impact

You will receive a response within 48 hours.

## Security Considerations

This tool reads data from `~/.claude/projects/`. This directory may contain:
- Conversation history with sensitive content
- File paths and code snippets
- Tool execution logs

Users should be aware that `ch` outputs this data to stdout and should use
appropriate caution when piping output or using `--json` mode.
