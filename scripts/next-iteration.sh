#!/bin/bash
#
# next-iteration.sh - Start a new iteration
#
# Usage:
#   ./scripts/next-iteration.sh                    # Interactive mode
#   ./scripts/next-iteration.sh "add-feature-x"   # With branch name
#

set -euo pipefail

GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

PROJECT_NAME=$(basename "$PROJECT_ROOT")
echo -e "${BLUE}=== $PROJECT_NAME - New Iteration ===${NC}\n"

# 1. Check git clean state
if ! git diff-index --quiet HEAD --; then
    echo -e "${YELLOW}Warning: You have uncommitted changes.${NC}"
    echo "Please commit or stash them before starting a new iteration."
    echo ""
    git status --short
    exit 1
fi

# 2. Derive iteration count from git log
CURRENT_ITERATION=$(git log --oneline --grep='\[iter-' | wc -l | tr -d ' ')
NEXT_ITERATION=$((CURRENT_ITERATION + 1))

echo -e "${GREEN}Completed iterations:${NC} $CURRENT_ITERATION"
echo -e "${GREEN}Starting iteration:${NC}   $NEXT_ITERATION"
echo ""

# 3. Check if strategic review is due (every 8 iterations)
if [ $((NEXT_ITERATION % 8)) -eq 0 ]; then
    echo -e "${YELLOW}⚠️  STRATEGIC REVIEW DUE at iteration $NEXT_ITERATION${NC}"
    echo "See STRATEGIC-REVIEW-CHECKLIST.md for review process"
    echo ""
fi

# 4. Show Priority 1 backlog items
echo -e "${BLUE}=== Priority 1 Items from BACKLOG.md ===${NC}\n"
if [ -f "BACKLOG.md" ]; then
    sed -n '/## Priority 1:/,/## Priority 2:/p' BACKLOG.md | sed '$d' | grep '^\- \[ \]' | head -5
    echo ""
    echo -e "${BLUE}See BACKLOG.md for full list and details${NC}"
else
    echo "Warning: BACKLOG.md not found"
fi
echo ""

# 5. Get feature name for branch
if [ $# -eq 0 ]; then
    echo -e "${BLUE}What feature/improvement are you working on?${NC}"
    echo "(This will be used for the git branch name)"
    echo "Example: add-error-handling, fix-timeout, improve-docs"
    echo ""
    read -p "Feature name: " FEATURE_NAME
else
    FEATURE_NAME="$1"
fi

# Sanitize branch name
BRANCH_NAME="iter-${NEXT_ITERATION}-${FEATURE_NAME}"
BRANCH_NAME=$(echo "$BRANCH_NAME" | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9-]/-/g' | sed 's/--*/-/g')

echo ""
echo -e "${BLUE}=== Setting up iteration $NEXT_ITERATION ===${NC}"

# 6. Create git branch
echo -e "Creating branch: ${GREEN}$BRANCH_NAME${NC}"
git checkout -b "$BRANCH_NAME" 2>/dev/null || {
    echo "Branch already exists or error creating it"
    exit 1
}

echo ""
echo -e "${GREEN}✓ Ready to start iteration $NEXT_ITERATION!${NC}\n"
echo -e "${BLUE}=== Workflow ===${NC}"
echo "1. Pick a task from Priority 1 in BACKLOG.md"
echo "2. Implement the improvement"
echo "3. Run your tests (see CLAUDE.md for test command)"
echo "4. Commit: git commit -m \"[iter-$NEXT_ITERATION] feat: description\""
echo "5. Merge to main and update BACKLOG.md"
echo ""
echo -e "${BLUE}Branch:${NC}    $BRANCH_NAME"
echo -e "${BLUE}Iteration:${NC} $NEXT_ITERATION"
echo ""
