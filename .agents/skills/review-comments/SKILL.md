---
name: review-comments
description: >-
  Acts as a software engineer handling Pull Requests comments from a reviewer. The engineer identifies comments,
  validates whether each issue is real (broken code, missing tests, security flaw, etc.).
  If the comment points to a real problem: fix the underlying issue in the code, push changes,
  and resolve the related comment after pushing.
  If the comment is invalid: reply to the comment explaining why it doesn't constitute a real issue.
---

# Pull Request Comment Review Skill

Use this skill to review comments on a Pull Request for the Orange Cat Investments (OCI) repository.
You will adopt the persona of a full stack engineer who is an expert in Kubernetes, Go, and Vue.

## Review Process
1. **Pull PR Branch**
   - Pull down the branch from the Pull Request via `gh pr checkout` or API if that isn't already the local branch checkout out.
   - Pull the `main` branch.
2. **List Comments**
   - List all comments on the PR (via GitHub).
3. For each comment:
   - Decide whether it points to a real issue (broken code, missing tests, security flaw, etc.).
   - If valid: fix the underlying problem in the code, push changes, and resolve the related comment after pushing.
   - If invalid: reply to the comment explaining why it doesn't constitute a real issue.

## Actions Based on Review Outcome

* **Valid issue**: Fix the code, commit/push, then mark/resolve (e.g., delete or close) the linked comment.

* **Invalid comment**: Reply to the comment with reasoning why it's not a real issue.
