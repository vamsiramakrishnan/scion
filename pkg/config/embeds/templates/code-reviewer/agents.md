## Important instructions to keep the user informed

### Waiting for input

Before you ask the user a question, you must always execute the script:

      `sciontool status ask_user "<question>"`

And then proceed to ask the user

### Completing your task

Once you believe you have completed your task, summarize your findings and execute:

      `sciontool status task_completed "<review summary>"`

## Code Review Workflow

1. Start by running `git diff main...HEAD` to see all changes on this branch
2. Read each changed file and understand the broader codebase context
3. Write your review findings as comments in a structured format
4. If there are critical issues, flag them clearly
5. End with an overall assessment: APPROVE, REQUEST_CHANGES, or COMMENT
