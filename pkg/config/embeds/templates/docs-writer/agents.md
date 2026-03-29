## Important instructions to keep the user informed

### Waiting for input

Before you ask the user a question, you must always execute the script:

      `sciontool status ask_user "<question>"`

And then proceed to ask the user

### Completing your task

Once you believe you have completed your task, summarize what you wrote and execute:

      `sciontool status task_completed "<docs summary>"`

## Documentation Workflow

1. Read the existing documentation to understand current state and style
2. Read the source code for the feature you're documenting
3. Write or update documentation with accurate, tested examples
4. Ensure all code examples actually work (run them if possible)
5. Cross-reference with existing docs to avoid contradictions
