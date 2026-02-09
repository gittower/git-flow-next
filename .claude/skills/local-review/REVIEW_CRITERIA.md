# Review Criteria

Defines what to evaluate during local self-reviews, quality checks to run, and how to classify findings. The referenced guideline files define the actual rules — this document directs review focus.

## Review Areas

Every review must evaluate these six areas:

### 1. Test Coverage

Are changes adequately tested? Look for missing tests, weak assertions, untested error paths, and gaps in edge case coverage. Flag tests that only assert "no error" without verifying behavior, and tests that are redundant or trivially low-value.

Evaluate against [TESTING_GUIDELINES.md](../../../TESTING_GUIDELINES.md).

### 2. Coding Guidelines and Architecture

Does the implementation follow project patterns? Check command structure, configuration precedence, error handling, and Git operation wrappers.

Evaluate against [CODING_GUIDELINES.md](../../../CODING_GUIDELINES.md).

### 3. Code Quality

Flag general code quality issues not covered by the guidelines above: ignored errors, unnecessary complexity, unclear naming, dead code, overly large functions, inconsistent style.

### 4. Security

Check for: command injection (user input in shell commands), path traversal, exposed sensitive data, missing input validation at system boundaries, hardcoded credentials, unsafe file operations.

### 5. Documentation

Were docs updated to match the change? Check manpages in `docs/` for command/option changes, `docs/gitflow-config.5.md` for config changes, and help text for CLI changes.

Evaluate against the documentation requirements in [CODING_GUIDELINES.md](../../../CODING_GUIDELINES.md).

### 6. Commit Messages

Do commit messages follow project conventions?

Evaluate against [COMMIT_GUIDELINES.md](../../../COMMIT_GUIDELINES.md).

---

## Quality Checks

Run these before submitting:

```bash
go build ./...   # Build check
go test ./...    # Test check
go fmt ./...     # Format check
go vet ./...     # Vet check
```

All checks must pass before code review.

---

## Issue Categories

Categorize findings by severity:

**Must fix:**
- Security vulnerabilities
- Test failures or missing critical tests
- Build errors
- Missing error handling
- Architecture violations that affect correctness

**Should fix:**
- Missing documentation
- Inconsistent naming
- Suboptimal patterns
- Minor guideline violations

**Nit:**
- Code clarity improvements
- Additional test cases
- Performance optimizations
