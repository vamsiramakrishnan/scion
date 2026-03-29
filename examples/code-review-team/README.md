# Example: Code Review Team

Launch a team of specialized agents to review your codebase from different angles simultaneously.

## What This Does

Starts 3 agents in parallel:
- **code-reviewer** — Reviews for bugs, logic errors, and code quality
- **security-reviewer** — Audits for vulnerabilities (OWASP Top 10)
- **docs-writer** — Checks documentation accuracy and coverage

Each agent gets its own git branch and works independently.

## Prerequisites

```bash
scion quickstart  # If not already set up
cd your-project   # Navigate to the project to review
scion init        # Initialize grove if needed
```

## Run

```bash
# Start all three agents
scion start reviewer "Review all recent changes for bugs and code quality issues" \
  --type code-reviewer

scion start security "Perform a security audit of the entire codebase" \
  --type security-reviewer

scion start docs "Review and improve the project documentation" \
  --type docs-writer

# Watch them work
scion list

# Check on individual agents
scion attach reviewer     # See the code reviewer's terminal
scion logs security       # Stream security audit progress
scion attach docs         # Check docs writer
```

## Review Results

Each agent works on its own branch. When they finish:

```bash
# See what each agent changed
# (Requires git diff API — or attach and run git diff manually)
scion attach reviewer
# Inside: git diff main...HEAD

# Merge the changes you want to keep
git merge reviewer-branch
git merge docs-branch
```

## Cleanup

```bash
scion delete reviewer --preserve-branch
scion delete security --preserve-branch
scion delete docs --preserve-branch
```
