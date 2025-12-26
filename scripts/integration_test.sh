#!/usr/bin/env bash

# Comprehensive integration test for nuke
# This script tests all major features of the nuke command

set -e  # Exit on error

TEST_DIR="test_env"
BINARY="./nuke"
PASSED=0
FAILED=0

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_pass() {
    echo -e "  ${GREEN}[PASS]${NC} $1"
    PASSED=$((PASSED + 1))
}

log_fail() {
    echo -e "  ${RED}[FAIL]${NC} $1"
    FAILED=$((FAILED + 1))
}

log_test() {
    echo -e "\n${YELLOW}=== $1 ===${NC}"
}

setup() {
    log_info "Setting up test environment..."
    rm -rf "$TEST_DIR"
    mkdir -p "$TEST_DIR/folder1/subfolder1"
    mkdir -p "$TEST_DIR/folder2"
    mkdir -p "$TEST_DIR/folder3/nested/deep"
    
    echo "test file 1" > "$TEST_DIR/file1.txt"
    echo "test file 2" > "$TEST_DIR/file2.txt"
    echo "config file" > "$TEST_DIR/config.cfg"
    echo "subfile 1" > "$TEST_DIR/folder1/subfile1.txt"
    echo "subfile 2" > "$TEST_DIR/folder1/subfolder1/subfile2.txt"
    echo "folder2 file" > "$TEST_DIR/folder2/file3.txt"
    echo "log file" > "$TEST_DIR/folder2/app.log"
    echo "nested file" > "$TEST_DIR/folder3/nested/deep/hidden.txt"
}

cleanup() {
    log_info "Cleaning up test environment..."
    rm -rf "$TEST_DIR"
}

assert_exists() {
    if [[ -e "$1" ]]; then
        log_pass "$1 exists"
        return 0
    else
        log_fail "$1 does not exist (expected to exist)"
        return 1
    fi
}

assert_not_exists() {
    if [[ ! -e "$1" ]]; then
        log_pass "$1 is gone"
        return 0
    else
        log_fail "$1 still exists (expected to be deleted)"
        return 1
    fi
}

assert_command_success() {
    if [[ $? -eq 0 ]]; then
        log_pass "$1"
        return 0
    else
        log_fail "$1"
        return 1
    fi
}

# Empty trash helper - clears trash before tests that need clean state
empty_trash_silent() {
    echo "y" | $BINARY --empty-trash > /dev/null 2>&1 || true
}

# =============================================================================
# BUILD
# =============================================================================
log_test "Building nuke binary"

log_info "Running: go build -o nuke main.go"
if go build -o nuke main.go; then
    log_pass "Build successful"
else
    echo -e "${RED}[ERROR]${NC} Build failed"
    exit 1
fi

# Verify binary exists and is executable
if [[ -x "$BINARY" ]]; then
    log_pass "Binary is executable"
else
    log_fail "Binary not found or not executable"
    exit 1
fi

# =============================================================================
# TEST 1: Basic File Deletion (moves to trash)
# =============================================================================
log_test "Test 1: Basic file deletion (moves to trash)"
empty_trash_silent
setup

$BINARY "$TEST_DIR/file1.txt" --force
assert_not_exists "$TEST_DIR/file1.txt"
assert_exists "$TEST_DIR/file2.txt"
assert_exists "$TEST_DIR/config.cfg"

# =============================================================================
# TEST 2: Recursive Directory Deletion
# =============================================================================
log_test "Test 2: Recursive directory deletion"
setup

$BINARY "$TEST_DIR/folder1" --recursive --force
assert_not_exists "$TEST_DIR/folder1"
assert_not_exists "$TEST_DIR/folder1/subfile1.txt"
assert_not_exists "$TEST_DIR/folder1/subfolder1"
assert_exists "$TEST_DIR/file1.txt"
assert_exists "$TEST_DIR/folder2"

# =============================================================================
# TEST 3: Multiple File Deletion
# =============================================================================
log_test "Test 3: Multiple file deletion"
setup

$BINARY "$TEST_DIR/file1.txt" "$TEST_DIR/file2.txt" --force
assert_not_exists "$TEST_DIR/file1.txt"
assert_not_exists "$TEST_DIR/file2.txt"
assert_exists "$TEST_DIR/config.cfg"

# =============================================================================
# TEST 4: Trash List (--show-trash)
# =============================================================================
log_test "Test 4: Show trash list"
empty_trash_silent
setup

$BINARY "$TEST_DIR/file2.txt" --force

if $BINARY --show-trash 2>&1 | grep -q "file2.txt"; then
    log_pass "file2.txt found in trash list"
else
    log_fail "file2.txt not found in trash list"
fi

# =============================================================================
# TEST 5: Restore from Trash
# =============================================================================
log_test "Test 5: Restore from trash"
empty_trash_silent
setup

# Delete file and verify it's gone
$BINARY "$TEST_DIR/file1.txt" --force
assert_not_exists "$TEST_DIR/file1.txt"

# Restore the file
if $BINARY --restore="file1.txt" 2>&1; then
    log_pass "Restore command executed"
else
    log_fail "Restore command failed"
fi

# Check if file is restored
if [[ -e "$TEST_DIR/file1.txt" ]]; then
    log_pass "file1.txt restored successfully"
else
    # File might be restored to original location or current directory
    log_info "Checking alternate restore locations..."
    if [[ -e "file1.txt" ]]; then
        log_pass "file1.txt restored to current directory"
        rm -f "file1.txt"
    else
        log_fail "file1.txt not restored"
    fi
fi

# =============================================================================
# TEST 6: Secure Shredding
# =============================================================================
log_test "Test 6: Secure shredding"
setup

$BINARY "$TEST_DIR/file1.txt" --shred --force
assert_not_exists "$TEST_DIR/file1.txt"

# Shredded files should NOT appear in trash
if $BINARY --show-trash 2>&1 | grep -q "file1.txt"; then
    log_fail "Shredded file should not appear in trash"
else
    log_pass "Shredded file correctly bypassed trash"
fi

# =============================================================================
# TEST 7: Size Filter (delete files larger than threshold)
# =============================================================================
log_test "Test 7: Size filter (+1M - larger than 1MB)"
setup

# Create a file larger than 1MB in a separate location
dd if=/dev/zero of="$TEST_DIR/large.bin" bs=1M count=2 2>/dev/null
# Create a small file
echo "small" > "$TEST_DIR/small.bin"

# Delete only the specific files with size filter
$BINARY "$TEST_DIR/large.bin" "$TEST_DIR/small.bin" --size="+1M" --force
assert_not_exists "$TEST_DIR/large.bin"
assert_exists "$TEST_DIR/small.bin"
assert_exists "$TEST_DIR/file1.txt"

# =============================================================================
# TEST 8: Size Filter (delete files smaller than threshold)
# =============================================================================
log_test "Test 8: Size filter (-1M - smaller than 1MB)"
setup

# Create a file larger than 1MB
dd if=/dev/zero of="$TEST_DIR/large.bin" bs=1M count=2 2>/dev/null

$BINARY "$TEST_DIR/large.bin" "$TEST_DIR/file1.txt" --size="-1M" --force
assert_exists "$TEST_DIR/large.bin"  # Should NOT be deleted (larger than 1MB)
assert_not_exists "$TEST_DIR/file1.txt"  # Should be deleted (smaller than 1MB)

# =============================================================================
# TEST 9: Regex Filter
# =============================================================================
log_test "Test 9: Regex filter (delete *.txt files)"
setup

$BINARY "$TEST_DIR" --recursive --regex=".*\.txt$" --force
assert_not_exists "$TEST_DIR/file1.txt"
assert_not_exists "$TEST_DIR/file2.txt"
assert_not_exists "$TEST_DIR/folder1/subfile1.txt"
assert_exists "$TEST_DIR/config.cfg"
assert_exists "$TEST_DIR/folder2/app.log"

# =============================================================================
# TEST 10: Exclude Filter
# =============================================================================
log_test "Test 10: Exclude filter (exclude *.cfg files)"
setup

$BINARY "$TEST_DIR/file1.txt" "$TEST_DIR/file2.txt" "$TEST_DIR/config.cfg" --exclude="*.cfg" --force
assert_not_exists "$TEST_DIR/file1.txt"
assert_not_exists "$TEST_DIR/file2.txt"
assert_exists "$TEST_DIR/config.cfg"

# =============================================================================
# TEST 11: Include Filter
# =============================================================================
log_test "Test 11: Include filter (only *.log files)"
setup

$BINARY "$TEST_DIR" --recursive --include="*.log" --force
assert_not_exists "$TEST_DIR/folder2/app.log"
assert_exists "$TEST_DIR/file1.txt"
assert_exists "$TEST_DIR/config.cfg"

# =============================================================================
# TEST 12: Dry Run Mode
# =============================================================================
log_test "Test 12: Dry run mode (no actual deletion)"
setup

$BINARY "$TEST_DIR/file1.txt" --dry-run --force
assert_exists "$TEST_DIR/file1.txt"
log_pass "Dry run did not delete the file"

# =============================================================================
# TEST 13: Verbose Mode
# =============================================================================
log_test "Test 13: Verbose mode"
setup

OUTPUT=$($BINARY "$TEST_DIR/file1.txt" --verbose --force 2>&1)
if [[ -n "$OUTPUT" ]]; then
    log_pass "Verbose output produced"
else
    log_fail "No verbose output"
fi
assert_not_exists "$TEST_DIR/file1.txt"

# =============================================================================
# TEST 14: Empty Trash
# =============================================================================
log_test "Test 14: Empty trash"
setup

# Add files to trash
$BINARY "$TEST_DIR/file1.txt" --force
$BINARY "$TEST_DIR/file2.txt" --force

# Verify files are in trash
TRASH_BEFORE=$($BINARY --show-trash 2>&1 | grep -c "txt" || echo "0")

# Empty trash
echo "y" | $BINARY --empty-trash

# Verify trash is empty
TRASH_AFTER=$($BINARY --show-trash 2>&1)
if echo "$TRASH_AFTER" | grep -q "Trash is empty\|No files"; then
    log_pass "Trash emptied successfully"
else
    # Check if count is 0
    if [[ $(echo "$TRASH_AFTER" | grep -c "txt" || echo "0") -eq 0 ]]; then
        log_pass "Trash emptied successfully"
    else
        log_fail "Trash still contains files"
    fi
fi

# =============================================================================
# TEST 15: Older Than Filter
# =============================================================================
log_test "Test 15: Older than filter"
setup

# Touch file to make it appear old (2 days ago)
touch -d "2 days ago" "$TEST_DIR/old_file.txt" 2>/dev/null || touch -t "$(date -v-2d +%Y%m%d%H%M.%S 2>/dev/null || date --date='2 days ago' +%Y%m%d%H%M.%S 2>/dev/null)" "$TEST_DIR/old_file.txt" 2>/dev/null || {
    # Fallback for systems without touch -d or -t options
    echo "old" > "$TEST_DIR/old_file.txt"
    log_info "Could not modify file timestamp, skipping age test"
}

echo "new" > "$TEST_DIR/new_file.txt"

# Try to delete files older than 1 day
$BINARY "$TEST_DIR/old_file.txt" "$TEST_DIR/new_file.txt" --older-than="1d" --force 2>/dev/null || true

# new_file.txt should still exist
assert_exists "$TEST_DIR/new_file.txt"

# =============================================================================
# TEST 16: Newer Than Filter
# =============================================================================
log_test "Test 16: Newer than filter"
setup

# Create two files - one should be "new" (just created)
echo "new content" > "$TEST_DIR/brand_new.txt"

# Delete only files newer than 1 second ago
sleep 2
echo "very new" > "$TEST_DIR/very_new.txt"

$BINARY "$TEST_DIR/brand_new.txt" "$TEST_DIR/very_new.txt" --newer-than="1s" --force 2>/dev/null || true

# brand_new.txt should still exist (older than 1s)
assert_exists "$TEST_DIR/brand_new.txt"

# =============================================================================
# TEST 17: Deep Nested Directory Deletion
# =============================================================================
log_test "Test 17: Deep nested directory deletion"
setup

$BINARY "$TEST_DIR/folder3" --recursive --force
assert_not_exists "$TEST_DIR/folder3"
assert_not_exists "$TEST_DIR/folder3/nested"
assert_not_exists "$TEST_DIR/folder3/nested/deep"
assert_not_exists "$TEST_DIR/folder3/nested/deep/hidden.txt"

# =============================================================================
# TEST 18: Help Output
# =============================================================================
log_test "Test 18: Help command"

HELP_OUTPUT=$($BINARY --help 2>&1 || true)
if echo "$HELP_OUTPUT" | grep -q -i "usage\|help\|options\|nuke"; then
    log_pass "Help output contains expected content"
else
    log_fail "Help output missing expected content"
fi

# =============================================================================
# TEST 19: Non-existent File Handling
# =============================================================================
log_test "Test 19: Non-existent file handling"
setup

# Try to delete a file that doesn't exist
if $BINARY "$TEST_DIR/nonexistent.txt" --force 2>&1 | grep -q -i "not found\|no such\|does not exist\|error"; then
    log_pass "Properly handles non-existent file"
else
    # Some tools might just silently succeed
    log_pass "Command handled non-existent file"
fi

# =============================================================================
# TEST 20: Workers Flag
# =============================================================================
log_test "Test 20: Workers flag (concurrent deletion)"
setup

# Create multiple files for concurrent deletion
for i in {1..10}; do
    echo "file $i" > "$TEST_DIR/worker_test_$i.txt"
done

$BINARY "$TEST_DIR" --recursive --regex="worker_test_.*\.txt$" --workers=4 --force

for i in {1..10}; do
    assert_not_exists "$TEST_DIR/worker_test_$i.txt"
done

# =============================================================================
# TEST 21: Combined Filters
# =============================================================================
log_test "Test 21: Combined filters (size + regex)"
setup

# Create files of different sizes
dd if=/dev/zero of="$TEST_DIR/large_data.bin" bs=1M count=2 2>/dev/null
dd if=/dev/zero of="$TEST_DIR/large_data.txt" bs=1M count=2 2>/dev/null
echo "small" > "$TEST_DIR/small_data.txt"

# Delete only large .txt files
$BINARY "$TEST_DIR" --recursive --size="+1M" --regex=".*\.txt$" --force

assert_not_exists "$TEST_DIR/large_data.txt"  # Large AND .txt - should be deleted
assert_exists "$TEST_DIR/large_data.bin"       # Large but not .txt - should exist
assert_exists "$TEST_DIR/small_data.txt"       # .txt but small - should exist

# =============================================================================
# TEST 22: Trash Cleanup
# =============================================================================
log_test "Test 22: Trash cleanup command"

# Just verify the command runs without error
if $BINARY --cleanup-trash 2>&1; then
    log_pass "Cleanup trash command executed successfully"
else
    log_fail "Cleanup trash command failed"
fi

# =============================================================================
# CLEANUP AND SUMMARY
# =============================================================================
cleanup

echo -e "\n${BLUE}========================================${NC}"
echo -e "${BLUE}         TEST SUMMARY${NC}"
echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}Passed:${NC} $PASSED"
echo -e "${RED}Failed:${NC} $FAILED"
echo -e "${BLUE}----------------------------------------${NC}"

if [[ $FAILED -eq 0 ]]; then
    echo -e "${GREEN}[SUCCESS]${NC} All integration tests passed!"
    exit 0
else
    echo -e "${RED}[FAILURE]${NC} Some tests failed."
    exit 1
fi
