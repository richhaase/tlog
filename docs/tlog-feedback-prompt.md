# tlog Feedback Prompt

Use this prompt to collect feedback from AI agents after they use tlog in a session.

---

## Prompt

You just used `tlog`, a task tracking CLI designed **specifically for AI coding agents**. You are the primary user, not humans.

**Design principles:**

- Unix philosophy — task management, nothing else
- Keep it simple — resist feature bloat
- Labels are markers — conventions without enforcement
- No agent identity — track tasks, not who's working on them

**Based on your actual usage this session, provide feedback:**

1. What commands did you use? What worked well?
2. Any friction or confusion?
3. Did you use `claim` before starting work? Why or why not?
4. Were dependencies/blocking relationships useful or overhead?
5. Is there a command you wished existed?
6. Anything that felt unnecessary?

Be specific and grounded in what you actually did — we value real friction over hypothetical features. Keep it brief.
