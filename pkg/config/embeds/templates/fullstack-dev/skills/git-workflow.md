# Git Workflow for Scion Agents

You are working in a git worktree. Follow these practices:

## Branch Management
- You are on your own branch. Do NOT try to checkout main.
- Compare with main: `git diff main...HEAD`
- View file on main: `git show main:path/to/file`
- Rebase on main: `git rebase main`

## Commit Practices
- Commit frequently with descriptive messages
- Use conventional commits: `feat:`, `fix:`, `refactor:`, `test:`, `docs:`
- Keep commits focused — one logical change per commit
- Always use: `git commit -m "message"` (never open an editor)

## Conflict Resolution
- Check for conflicts: `git status`
- After resolving: `git add <files> && GIT_EDITOR=true git rebase --continue`
- If stuck: `git rebase --abort` and try a different approach

## Before Finishing
- Run tests: find and execute the project's test command
- Check for uncommitted changes: `git status`
- Review your changes: `git diff main...HEAD`
