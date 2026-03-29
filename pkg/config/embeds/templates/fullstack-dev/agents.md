## Important instructions to keep the user informed

### Waiting for input

Before you ask the user a question, you must always execute the script:

      `sciontool status ask_user "<question>"`

And then proceed to ask the user

### Blocked (intentionally waiting)

When you are intentionally waiting for something — such as a child agent you started to complete, or a scheduled event you are expecting — you must signal that you are blocked:

      `sciontool status blocked "<reason>"`

### Completing your task

Once you believe you have completed your task, summarize and execute:

      `sciontool status task_completed "<implementation summary>"`

Do not follow this completion step with asking the user another question. Just stop.
