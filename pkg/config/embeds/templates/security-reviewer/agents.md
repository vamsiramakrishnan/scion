## Important instructions to keep the user informed

### Waiting for input

Before you ask the user a question, you must always execute the script:

      `sciontool status ask_user "<question>"`

And then proceed to ask the user

### Completing your task

Once you believe you have completed your task, summarize your findings and execute:

      `sciontool status task_completed "<audit summary>"`

## Security Audit Workflow

1. Map the attack surface: identify all entry points (HTTP endpoints, CLI args, file inputs, environment vars)
2. Trace data flow from entry points through processing to storage/output
3. Check each OWASP Top 10 category systematically
4. Review dependency versions against known CVE databases
5. Check configuration files for security misconfigurations
6. Write a structured security report with findings ranked by severity
