# Strategic Review Checklist

**Purpose:** Guide the strategic review process that happens every 8 iterations.

**When to use:** At iterations 8, 16, 24, 32, etc.

**Time required:** 45-60 minutes

**Last Review:** N/A (update when you run your first review)

---

## Overview

Strategic reviews are critical checkpoints to ensure you're:
- ✅ Building the right things (not just building things right)
- ✅ Climbing the right hill (avoiding local maxima)
- ✅ Maintaining code quality and architecture
- ✅ Staying motivated and making meaningful progress

**This review prevents:**
- ❌ Drifting off-course due to accumulated small decisions
- ❌ Technical debt accumulation
- ❌ Building features nobody needs
- ❌ Burnout from loss of purpose

---

## Pre-Review Preparation (5 minutes)

Before starting the review, gather these materials:

- [ ] **Git log** - Last 8 iterations of commits
  ```bash
  git log --oneline --since="$(git log --reverse --format=%ct | head -8 | tail -1 | xargs -I {} date -r {} +%Y-%m-%d)"
  ```

- [ ] **Test status** - Confirm all tests passing
  ```bash
  # Run your test command (see CLAUDE.md for this project's test command)
  ```

- [ ] **Metrics** - Count features, tests, docs added since last review

- [ ] **User feedback** - Any questions, issues, or feature requests received

---

## Part 1: Direction Check (15 minutes)

**Goal:** Confirm you're climbing the right hill.

### Questions to Answer:

1. **Are we solving the right problem?**
   - [ ] Review original project goals from README.md
   - [ ] Has the problem statement changed?
   - [ ] Are we still focused on the core mission?
   - [ ] Decision: Continue current direction / Pivot / Refocus

2. **What patterns emerged from the last 8 iterations?**
   - [ ] Which types of improvements were most common? (features/docs/tests/refactoring)
   - [ ] Which took longer than expected?
   - [ ] Which provided most value?
   - [ ] Which were most enjoyable?

3. **What user needs are still unmet?**
   - [ ] Review "What's Missing" section in STRATEGIC-ANALYSIS.md
   - [ ] Any new pain points discovered?
   - [ ] What questions do users keep asking?
   - [ ] What workarounds are people using?

4. **Hill climbing check:**
   - [ ] Are we making progress toward the goal? (not just spinning)
   - [ ] Are we going up the right hill? (not optimizing wrong thing)
   - [ ] Any signs of diminishing returns?

**Outcome:** Write 2-3 sentences on current direction health.

---

## Part 2: Backlog Health Check (10 minutes)

**Goal:** Ensure backlog is useful and current.

### Review BACKLOG.md:

1. **Priority 1 Items**
   - [ ] Still high-value, small-scope?
   - [ ] Still relevant to current direction?
   - [ ] Any stuck too long (5+ iterations without progress)?
   - [ ] Action: Keep / Demote / Delete stuck items

2. **Priority 2-3 Items**
   - [ ] Anything that should be promoted to Priority 1?
   - [ ] Anything that's no longer relevant?
   - [ ] Action: Promote urgent items / Delete obsolete items

3. **Ideas Inbox**
   - [ ] How many unsorted ideas? (> 10 means backlog grooming overdue)
   - [ ] Sort inbox items into priority sections
   - [ ] Archive or delete obviously bad ideas

4. **Completed Section**
   - [ ] Move last 8 iterations' work to Completed
   - [ ] Celebrate wins! (This is important for motivation)

**Outcome:** Clean, prioritized backlog ready for next 8 iterations.

---

## Part 3: Code Health Assessment (10 minutes)

**Goal:** Identify architectural friction and technical debt.

### Questions to Answer:

1. **Test Health**
   - [ ] All tests passing? (Must be YES)
   - [ ] Test count: _____ (should be increasing)
   - [ ] Any flaky tests?
   - [ ] Coverage gaps in new features?

2. **Architectural Friction**
   - [ ] Any features that were harder to add than expected? Why?
   - [ ] Any code that feels messy or confusing?
   - [ ] Any patterns being repeated that should be abstracted?
   - [ ] Is the architecture still clean?

3. **Technical Debt**
   - [ ] Any TODOs or FIXMEs in the code?
   - [ ] Any workarounds that should be proper solutions?
   - [ ] Any deprecated patterns still in use?
   - [ ] Any performance issues noticed?

4. **Documentation Accuracy**
   - [ ] README.md still accurate?
   - [ ] Examples still work?
   - [ ] QUICKREF.md up to date?
   - [ ] Comments still correct?

**Outcome:** List of 0-3 technical debt items to add to backlog.

---

## Part 4: Process Reflection (10 minutes)

**Goal:** Improve the development process itself.

### Questions to Answer:

1. **Iteration Discipline**
   - [ ] Average iteration time? (Should be 30-60 min)
   - [ ] Any iterations that went over scope? Why?
   - [ ] Were constraints respected? (No failing test commits, scope limits, etc.)
   - [ ] Did TodoWrite tool help? Used consistently?

2. **Documentation Workflow**
   - [ ] Were docs updated as features were added?
   - [ ] Is CLAUDE.md helpful?
   - [ ] Is BACKLOG.md being used effectively?
   - [ ] Is this STRATEGIC-REVIEW-CHECKLIST.md useful?

3. **Joy & Sustainability**
   - [ ] Still enjoying the work?
   - [ ] Feeling motivated to continue?
   - [ ] Process feels smooth or burdensome?
   - [ ] Celebrating wins or just grinding?

4. **Tools & Automation**
   - [ ] Did scripts/next-iteration.sh save time?
   - [ ] Any manual steps that should be automated?
   - [ ] Any tools/processes that should be simplified?

**Outcome:** 0-2 process improvements to implement.

---

## Part 5: Next 8 Iterations Plan (10 minutes)

**Goal:** Set clear direction for the next review cycle.

### Planning Questions:

1. **Primary Focus Area** (pick one)
   - [ ] Features (expand capabilities)
   - [ ] Usability (make existing features easier)
   - [ ] Documentation (help users discover value)
   - [ ] Testing (increase reliability confidence)
   - [ ] Performance (speed up operations)
   - [ ] Architecture (reduce friction for future work)

2. **Success Criteria**
   - What would make the next 8 iterations successful?
   - What specific outcomes matter?
   - How will we know if we're on track?

3. **Backlog Priorities**
   - Pick top 3-5 items from Priority 1 as the focus
   - Mark them with [FOCUS] tag in BACKLOG.md
   - These are the next-iteration candidates

4. **Strategic Risks**
   - What could derail progress?
   - Any external dependencies or blockers?
   - Mitigation plans?

**Outcome:** Clear focus for next 8 iterations written in STRATEGIC-ANALYSIS.md

---

## Post-Review Actions (5 minutes)

Complete these final steps:

- [ ] **Update STRATEGIC-ANALYSIS.md** with review findings (date, key decisions, focus areas)
- [ ] **Update BACKLOG.md** "Last Updated" field
- [ ] **Update this checklist** "Last Review" field at top
- [ ] **Commit changes**: `git commit -m "docs: iteration X strategic review"`
- [ ] **Celebrate** - You just completed a thorough review! Take a break.

---

## Review Output Template

Use this template to document the review in STRATEGIC-ANALYSIS.md:

```markdown
## Strategic Review - Iteration X (YYYY-MM-DD)

### Direction
[2-3 sentences on direction health]

### Key Findings
- [Finding 1]
- [Finding 2]
- [Finding 3]

### Decisions Made
- [Decision 1]
- [Decision 2]

### Focus for Next 8 Iterations
Primary focus area: [AREA]
Success criteria: [CRITERIA]
Priority items: [3-5 BACKLOG ITEMS]

### Metrics
- Tests: X → Y (+Z)
- Features: X → Y (+Z)
- Docs: X pages
- Lines of code: X

### Process Improvements
- [Improvement 1]
- [Improvement 2]
```

---

## Red Flags to Watch For

These indicate the process is breaking down and needs immediate attention:

### Death March Warnings ⚠️
- **Making exceptions** - "I'll commit failing tests and fix later"
- **Loss of meaning** - Bumping iteration count without real value
- **Fighting the process** - Seeing tests/discipline as obstacle, not safety net
- **Scope creep** - Regular iterations taking 2-3 hours
- **Technical debt accumulation** - "I'll refactor it later" becoming a pattern

### Direction Problems ⛰️
- **Stuck on Priority 1 items** - Same item stuck for 5+ iterations (wrong priority or too large)
- **Empty Completed section** - Not celebrating wins leads to burnout
- **Inbox overflow** - 10+ unsorted ideas means backlog is not being maintained
- **Feature churn** - Building features that get removed or replaced quickly

### Code Quality Problems 🏗️
- **Flaky tests** - Tests occasionally failing without code changes
- **Avoiding testing** - Adding features without tests
- **Copy-paste code** - Same patterns repeated instead of abstracted
- **Documentation lag** - README outdated, examples broken

**If you see multiple red flags, STOP and address them before continuing iterations.**

---

## Notes on This Checklist

### First Time Using This?
Don't worry about perfection. The review will get faster and more effective with practice. Focus on answering the key questions honestly.

### Time Management
- Set a 60-minute timer
- If you're spending > 15 min on a section, you're going too deep
- The goal is strategic overview, not detailed analysis

### Frequency
Every 8 iterations is recommended. If iterations are very quick (< 30 min each), consider every 10-12 iterations. If iterations are longer (> 60 min), consider every 6 iterations.

### Adjusting This Checklist
This checklist serves you. If sections aren't useful, modify or remove them. If you find yourself asking the same question repeatedly, add it.

---

**Remember:** This review exists to keep you on track, not to create busywork. If it's not valuable, something is wrong with the review, not with you.
