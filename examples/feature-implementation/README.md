# Example: Feature Implementation Pipeline

A sequential pipeline where an architect designs a feature, then a developer implements it.

## What This Does

1. **Architect agent** analyzes requirements and creates an implementation plan
2. **Developer agent** implements the plan on a feature branch
3. **Reviewer agent** reviews the implementation

## Run

```bash
cd your-project
scion init  # if needed

# Step 1: Architecture
scion start architect \
  "Design a user authentication system with JWT tokens, refresh tokens, and role-based access control. Create a detailed implementation plan." \
  --type architect --attach

# Wait for architect to finish (it will report task_completed)
# Review the plan, then:

# Step 2: Implementation
scion start developer \
  "Implement the auth system based on the design in .design/ or the architect's branch. Follow the implementation plan exactly." \
  --type fullstack-dev --attach

# Step 3: Review
scion start reviewer \
  "Review the auth implementation for security issues, bugs, and code quality" \
  --type code-reviewer --attach
```

## Tips

- Each agent works on its own git branch automatically
- Use `scion message <name> "additional context"` to send follow-up instructions
- Use `scion list` to monitor all agents
- Use `--preserve-branch` when deleting to keep the work
