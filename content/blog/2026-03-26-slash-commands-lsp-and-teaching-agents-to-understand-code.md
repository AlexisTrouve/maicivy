---
title: "Slash Commands, LSP, and Teaching Agents to Understand Code"
date: 2026-03-26
meta_description: "How I built slash commands, integrated LSP for real code intelligence, and automated CLAUDE.md management in Melodicode this week."
tags: ["flutter", "dart", "lsp", "ai-agents", "developer-tools"]
draft: false
---

# Slash Commands, LSP, and Teaching Agents to Understand Code

Three features landed in Melodicode this week. Any one of them would have been a full week on its own. Together, they shift what the tool actually is.

I'll focus on the two that matter most: the slash commands system and LSP integration. They're connected in ways that weren't obvious at the start.

## The problem with text-only interfaces

Melodicode is an AI-orchestrated IDE. A coordinator agent breaks down tasks, spawns workers, manages the flow. The interface for interacting with all of this was, until now, a chat input. You type, agents respond.

The problem: as the system grows, the things you need to tell it grow too. Open project. Switch context. Save session. Load a workspace. Trigger a specific agent behavior. All of this ends up as natural language, which means the coordinator has to interpret intent before it can act. That's latency, token usage, and occasional ambiguity where there shouldn't be any.

Slash commands are the solution to this. Not because they're clever, but because they're precise.

## Building the slash commands system

The design goal was simple: typed commands that map directly to app behavior, bypassing the agent interpretation layer entirely for common operations.

A slash command like `/session save my-refactor` shouldn't route through the coordinator. It should hit a handler directly, execute, and confirm. The coordinator is for complex reasoning. Saving a session is not complex reasoning.

The UI side works like this: typing `/` in the input triggers a command palette overlay. As you continue typing, it filters available commands. Select one, fill any required arguments, execute. The pattern is familiar from Notion, Slack, VS Code — users don't need to learn it.

Under the hood, each command is a registered handler with a name, argument spec, and an execution function. Adding a new command means adding a handler — nothing else changes. The command registry stays decoupled from the UI layer.

```dart
final commands = [
  SlashCommand(
    name: 'session',
    subcommands: ['save', 'load', 'list'],
    handler: sessionCommandHandler,
  ),
  SlashCommand(
    name: 'agent',
    subcommands: ['spawn', 'kill', 'status'],
    handler: agentCommandHandler,
  ),
];
```

This connects directly to the session persistence plan that's been in progress. The `/session save`, `/session load`, and `/session list` commands are the user-facing surface for that entire persistence system. The backend serializes agent state, conversations, open projects. The slash command is the trigger.

What's interesting about this architecture: it makes the coordinator smarter by handling less. The coordinator should spend its reasoning budget on tasks that require reasoning. Navigation, state management, workspace switching — these are deterministic operations. Giving them a deterministic interface makes the whole system more predictable.

## LSP: the bigger unlock

If slash commands make the interface more precise, LSP makes the agents more capable.

The core issue I've been sitting with: agents in Melodicode can read files and write code. But they do it without understanding. They don't know if a function exists before calling it. They don't know if the types match. They don't see deprecation warnings. They write code the way someone might write in a language they've memorized but never compiled — syntactically plausible, semantically blind.

LSP (Language Server Protocol) is what IDEs use to give developers real-time code intelligence. Hover types, go-to-definition, diagnostics, completions. It's a client-server protocol — the editor is the client, a language-specific server (tsserver for TypeScript, rust-analyzer for Rust, the Dart analysis server for Dart) is the server.

Integrating LSP into Melodicode means agents become the client. They can query the language server the same way VS Code does.

The implementation runs one LSP server per project. More memory overhead than a shared server, but it gives clean isolation. A Dart project and a TypeScript project each get their own analysis context. No state bleed between projects.

Agents get three new tools: `lsp_diagnostics`, `lsp_hover`, and `lsp_goto_definition`. The usage pattern changes how they approach edits.

```dart
// Before editing a file
final diagnostics = await lspService.getDiagnostics(filePath);
if (diagnostics.isNotEmpty) {
  final errors = diagnostics.where(
    (d) => d.severity == DiagnosticSeverity.error
  );
  agent.report('${errors.length} errors in $filePath before edit');
}

// After editing
final postEditDiagnostics = await lspService.getDiagnostics(filePath);
if (postEditDiagnostics.length > diagnostics.length) {
  agent.report('Edit introduced new errors — reverting');
}
```

The protocol layer itself was the tricky part. LSP uses JSON-RPC over stdio, and the response parsing had fundamental issues that only surfaced in Round 4 of TDD. Seven categories of bugs, twenty failing tests — type coercion problems, response parsing edge cases, initialization sequence issues. None of these showed up in the first three rounds. The bugs were subtle enough that basic happy-path tests couldn't catch them.

The lesson here is that TDD round count matters. A protocol implementation that passes three rounds of tests can still have fundamental parsing bugs. Specifically around how the client handles server capabilities negotiation, how it parses notifications versus responses, and how it deals with partial reads on the stdio stream. These aren't the cases you write first.

## CLAUDE.md management

The third feature — CLAUDE.md management — is worth mentioning briefly because it solves a real friction point.

CLAUDE.md files are how you give Claude Code context about a project: architecture, conventions, what to avoid. They're powerful but manual. You write them once, they go stale, you forget to update them.

The feature automates the management layer: 6 commits, +797 lines, 5 files. Agents can now read, update, and maintain CLAUDE.md files as part of their normal operation. When a worker agent makes an architectural decision, it can record it. When conventions change, the file updates. The context stays current without requiring manual maintenance.

This connects to the broader direction: reduce the amount of manual coordination the human has to do. The Duo integration I built earlier (native background service for agent-to-agent communication) was the same impulse. The human shouldn't be a relay node.

## What this week actually represents

Looking at these three features together: slash commands, LSP, CLAUDE.md management — they're all removing friction from different layers.

Slash commands remove friction from the UI layer. Instead of describing what you want to do, you say it precisely.

LSP removes friction from the agent execution layer. Instead of trial-and-error code edits, agents check before they act.

CLAUDE.md management removes friction from the context layer. Instead of manually maintaining project documentation, agents keep it current.

The session persistence plan — still in progress across its six phases — sits underneath all of this. Agent state, notifications, logs, named workspaces. When that's done, closing and reopening Melodicode will be seamless. The groundwork is there.

One hundred commits across two projects this week. The number isn't the point. The point is that all three features push in the same direction: a system where the human makes decisions and the tooling handles everything else.

That's the actual goal.
