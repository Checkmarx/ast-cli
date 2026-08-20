# AI Governance Solution — PPT Presentation Guide

---

## 1. Importance of AI Coding Agent Governance — Why

- 96% of organizations are already running AI agents in production — but only 21% have a mature governance model for them.
- 53% of organizations have already experienced AI agents exceeding their intended permissions.
- 47% have faced a security incident involving an AI agent in the past 12 months.
- AI coding agents run with real privileges — filesystem access, shell execution, database and MCP tool access — yet most orgs have no real-time control over what they actually do once running.
- Regulation is arriving faster than governance maturity: EU AI Act, SOC2, ISO 42001, NIST AI RMF all now expect logged, reviewable, human-overseen AI decisions — most orgs can't produce that evidence today.
- The gap isn't awareness — it's enforcement. Everyone agrees governance matters; almost nobody has turned that into a runtime control.

*Sources: [Omnithium — AI Agent Compliance](https://omnithium.ai/blog/ai-agent-compliance-soc2-iso-eu-ai-act.html), [MiniOrange — AI Agent Compliance Challenges](https://www.miniorange.com/blog/ai-agent-compliance-challenges/)*

---

## 2. Competitors — Market Research (Endor Labs, Snyk)

**Both Endor Labs and Snyk already ship AI agent governance products — this is not a gap in the market, it's an active, fast-moving category.**

- **Snyk** launched *Agent Security* + *Evo Agentic Development Security* + GA of *Evo AI-SPM* at RSA Conference, March 2026. It enforces real-time policy on AI agents in the workflow — blocking destructive actions before they execute, governing what agents access and what workflows they run.
- **Endor Labs** ships *Agent Governance* (part of its AURI platform) — real-time visibility and policy enforcement over AI coding agents: block/allow/audit actions across prompts, commands, tools, and skills, and control access to sensitive files/directories.
- Both are materially similar in capability to what's described here: real-time BLOCK/ALLOW/AUDIT enforcement on agent behavior. This is a converging category, not a blue ocean.

**Wider market:** Independent research counts at least 8–11 distinct AI agent governance/security platforms today (Kosmoy's comparison, Linx Security's "Top 11 Agentic AI Security Solutions"), with dozens more in the adjacent agentic-AI-security space — Microsoft, Palo Alto Networks, CrowdStrike, Zenity, Noma, WitnessAI, Cisco AI Defense, Credo AI, Protect AI, HiddenLayer, Lakera, and others. This is a genuinely crowded, newly-formed category — most of these launches happened in 2026.

**Why Checkmarx wins here:**
- Already inside the workflow security teams trust — SAST, SCA, and ASCA are established, procured, and running today. AI agent governance becomes one more capability inside a platform teams already run, not a new vendor to evaluate, onboard, and integrate from scratch.
- Consolidation is a real, tangible cost and fatigue reducer — security teams are already stretched thin covering every new AI risk surface with a separate point tool; folding agent governance into an existing platform relationship is a concrete win, not a marketing line.
- Enforcement happens locally, at the hook, in milliseconds, fail-open by design — governance decisions never sit on the developer's hot path waiting on a network round trip, so security doesn't come at the cost of velocity.
- Every action is captured, not just the blocked ones — sessions, prompts, skills, and MCP tool usage are all recorded, so an investigation or audit is never limited to "what got flagged."
- Forensic-grade traceability — every decision is permanently tied to the exact policy version that produced it, so "why was this blocked six months and ten policy revisions ago" always has a real answer.
- Compliance evidence is a byproduct of enforcement, not a separate reporting exercise — the same audit trail that powers violations and reviews automatically satisfies SOC2 and EU AI Act evidence requirements, with no duplicate logging effort.

*Sources: [Snyk — Agent Security launch](https://snyk.io/news/snyk-launches-agent-security-solution/), [Snyk — Evo Agentic Development Security](https://snyk.io/news/snyk-launches-evo-agentic-development-security/), [Endor Labs — AI Coding Agent Governance](https://www.endorlabs.com/ai-coding-agent-governance), [Endor Labs — Designing Agent Governance](https://www.endorlabs.com/learn/designing-agent-governance-a-new-surface-for-ai-risk), [Kosmoy — Best AI Agent Governance Platforms 2026](https://www.kosmoy.com/resources/blog/best-ai-agent-governance-platforms-2026/), [Linx Security — Top 11 Agentic AI Security Solutions](https://www.linx.security/blog/top-agentic-ai-security-solutions), [Checkmarx — Redefines AppSec for Agentic Development](https://checkmarx.com/checkmarx-redefines-application-security-for-the-age-of-agentic-development/)*

---

## 3. Story — Pain Points

- A developer's AI agent reads a `.env` file and prints AWS credentials into a chat log — no one blocked it, no one was notified, no one reviewed it, and there's no record it ever happened.
- A security team is asked "what did this AI agent actually do across its entire session yesterday?" — and has no answer, because only violations were ever tracked, not the full sequence of prompts, tool calls, and decisions.
- A compliance team gets asked in an audit: "What controls exist on AI agent access to customer data, and can you prove a human reviewed the risky ones?" — and has no answer, because nothing was ever logged with that intent, and no oversight trail exists.
- A security team discovers, after the fact, that an agent invoked a Slack MCP tool and several custom skills no one approved — because there was no inventory of what capabilities and tools agents even had access to, let alone what they were doing with them.
- The common thread: this isn't about one policy that was missing. It's that nobody has a complete, trustworthy record of what AI agents did, why, and who — if anyone — looked at it. Governance without full activity tracking and auditing is just a rule that fires in the dark.

---

## 4. What Our Solution Provides — AI Governance

- **Real-time policy enforcement** — BLOCK / ALERT / ALLOW decisions on filesystem, shell, MCP tool, web search, and prompt-level actions, evaluated as the agent acts, not after the fact.
- **Violation tracking with full context** — every decision records who, what, which policy, and when — not just that something happened.
- **Human-in-the-loop oversight** — violations require a named reviewer, and segregation of duties is enforced server-side (a developer cannot review their own violation) — not a UI convention.
- **Skills & MCP tool discovery** — automatic inventory of what capabilities and third-party tools agents actually have access to and are invoking.
- **Third-party enforcement detection** — the platform detects other governance/enforcement tooling already present in an agent's session, so nothing is flying under the radar.
- **Immutable audit trail** — every policy lifecycle change and every agent action is logged and attributable to an actor.
- **Automatic policy-to-compliance mapping** — policies are mapped to the controls they satisfy (SOC2, EU AI Act) the moment they're created, with no manual tagging step.
- **Full policy lifecycle management** — versioned create/update/activate/disable/delete, every change attributed and audited.

---

## 5. Who Is the Audience

- **AppSec / Security Engineers** — want real-time detection and blocking, not after-the-fact scanning; full audit trail for incident response.
- **Compliance / Governance Teams** — need to prove, with evidence, that human oversight and logging obligations (EU AI Act, SOC2) are actually being met.
- **CISOs / Risk Leaders** — need to scale AI agent adoption without their risk exposure scaling faster than their controls.
- **Developers** — not the primary buyer or concern here; governance is designed to run in the background with minimal friction, so this audience matters mainly in that it should barely notice the platform is there.

---

## 6. Architecture Overview

```
   AI Coding Agent                  Every single action the
   (Claude, Cursor, etc.)           agent takes is watched
          │                          — nothing happens unseen
          ▼
   Continuous Oversight             Every action is checked
   (watches every action,     ──►   against company policy,
    in real time)                   the instant it happens
          │
          ▼
   Policy Decision                  Allowed, flagged for review,
   (Allow / Flag / Block)           or stopped outright —
          │                          automatically, instantly
          ▼
   Central Record & Evidence        Nothing is lost — every
   (secure, tamper-proof log)  ──►  decision becomes proof for
          │                          audits and investigations
          ▼
   Governance Dashboard             Security, compliance, and
   (for security & compliance)      leadership see it all, live
```

**In plain terms:** every action an AI coding agent takes is watched the moment it happens, automatically checked against company policy, and recorded permanently — so nothing slips through unnoticed, and there's always evidence to show what happened, why, and who reviewed it. This happens instantly, without slowing developers down.

---

## 7. Policies, Violations, MCP, Skills — Real-Time Governance & Auditing

- **Policies** are structured rules (type: filesystem/shell/mcp/web/prompt, action: BLOCK/ALERT, target) — not free text — so they can be reliably matched, audited, and mapped to compliance controls.
- **Violations** capture every triggered policy decision with full context (actor, action attempted, matching rule, timestamp) and stay open until a human reviews them — reviewing, not "fixing," since the action already happened.
- **MCP tool discovery** auto-inventories which external tools (Slack, GitHub, filesystem, web search, database) an agent can reach, so unapproved or unknown integrations don't go unnoticed.
- **Skills discovery** tracks which agent capabilities are actually being invoked, by whom, and how often — turning "what can this agent do" from a guess into an observed fact.
- **Third-party tool blocking/observation** — the platform also detects when other enforcement tooling is already active in a session, so governance isn't operating blind to what else is in place.
- All of this feeds the same audit trail used for compliance evidence — nothing is a separate system of record.

---

## 8. Compliance

- **SOC2** — controls with real, operational evidence today include access provisioning/deprovisioning (CC6.1, CC6.2/6.5), boundary protection (CC6.6), data transmission restriction (CC6.7), anomaly detection (CC7.1), security event review (CC7.2), change management (CC8.1), and third-party risk visibility (CC9.2).
- **EU AI Act** — deployer obligations under Art.26 (human oversight, monitoring, logging) and Art.12 (record-keeping) and Art.14 (human oversight of automated decisions) are backed by real evidence generated automatically from policy and review activity.
- Compliance status is reported as raw counts of controls (compliant / partial / non-compliant / planned) — never blended into a single score, because only an auditor can make that claim, not the platform.
- Every piece of compliance evidence traces back to the exact policy decision or human review that produced it — nothing is asserted without a record behind it.

---

## 9. Roadmap — Future Enhancements

- **Natural language policy creation** — let security teams describe intent in plain English ("block access to files containing credit card numbers") and have it translated into a structured, enforceable policy.
- **Extending hooks support to other agents** — beyond the current agent integration, to bring the same real-time enforcement model to other AI coding agents and IDEs.
- **Capturing user intent** — analyzing what a prompt is actually trying to achieve, not just what tool or file it touches. A file access or shell command can look identical on the surface whether it's routine work or someone probing for a prompt injection / jailbreak attempt — intent detection on the prompt itself is what tells those apart, and lets policy respond to the *why* behind an action, not just the *what*.
