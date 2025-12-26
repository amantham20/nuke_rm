#!/usr/bin/env fish

# Comprehensive integration test for nuke

set TEST_DIR "test_env"
set BINARY "./nuke"

# Colors
set RED (set_color red)
set GREEN (set_color green)
set YELLOW (set_color yellow)
set NC (set_color normal)

function setup
    echo "Setting up test environment..."
    rm -rf $TEST_DIR
    mkdir -p $TEST_DIR/folder1/subfolder1
    mkdir -p $TEST_DIR/folder2
    
    echo "test file 1" > $TEST_DIR/file1.txt
    echo "test file 2" > $TEST_DIR/file2.txt
    echo "subfile 1" > $TEST_DIR/folder1/subfile1.txt
    echo "subfile 2" > $TEST_DIR/folder1/subfolder1/subfile2.txt
    echo "folder2 file" > $TEST_DIR/folder2/file3.txt
end

function cleanup
    echo "Cleaning up..."
    rm -rf $TEST_DIR
end

function assert_exists
    if test -e $argv[1]
        printf "  %s[PASS]%s %s exists\n" "$GREEN" "$NC" "$argv[1]"
    else
        printf "  %s[FAIL]%s %s does not exist\n" "$RED" "$NC" "$argv[1]"
        exit 1
    end
end

function assert_not_exists
    if not test -e $argv[1]
        printf "  %s[PASS]%s %s is gone\n" "$GREEN" "$NC" "$argv[1]"
    else
        printf "  %s[FAIL]%s %s still exists\n" "$RED" "$NC" "$argv[1]"
        exit 1
    end
end

# Build the binary
echo "Building nuke..."
go build -o nuke main.go
if test $status -ne 0
    echo "$RED[ERROR]$NC Build failed"
    exit 1
end

# Test 1: Basic deletion (default is trash)
setup
echo "Test 1: Basic deletion of a file (moves to trash)"
$BINARY $TEST_DIR/file1.txt --force
assert_not_exists $TEST_DIR/file1.txt
assert_exists $TEST_DIR/file2.txt

# Test 2: Recursive deletion
setup
echo "Test 2: Recursive deletion of a folder"
$BINARY $TEST_DIR/folder1 --recursive --force
assert_not_exists $TEST_DIR/folder1
assert_exists $TEST_DIR/file1.txt

# Test 3: Trash list
setup
echo "Test 3: Trash list"
$BINARY $TEST_DIR/file2.txt --force
# Check if it's in trash (using nuke --show-trash)
$BINARY --show-trash | grep "file2.txt" > /dev/null
if test $status -eq 0
    printf "  %s[PASS]%s file2.txt found in trash list\n" "$GREEN" "$NC"
else
    printf "  %s[FAIL]%s file2.txt not found in trash list\n" "$RED" "$NC"
    exit 1
end

# Test 4: Restore from trash
echo "Test 4: Restore from trash"
# Empty trash first to have a clean state
printf "y\n" | $BINARY --empty-trash
setup
$BINARY $TEST_DIR/file2.txt --force
rm -f $TEST_DIR/file2.txt
$BINARY --restore="file2.txt"
assert_exists $TEST_DIR/file2.txt

# Test 5: Shredding
setup
echo "Test 5: Shredding a file"
$BINARY $TEST_DIR/file1.txt --shred --force
assert_not_exists $TEST_DIR/file1.txt

# Test 6: Size filter
setup
echo "Test 6: Size filter"
# Create a larger file
dd if=/dev/zero of=$TEST_DIR/large.bin bs=1M count=2 2>/dev/null
$BINARY $TEST_DIR --recursive --size="+1M" --force
assert_not_exists $TEST_DIR/large.bin
assert_exists $TEST_DIR/file1.txt

# Test 7: Regex filter
setup
echo "Test 7: Regex filter"
$BINARY $TEST_DIR --recursive --regex=".*\.txt" --force
assert_not_exists $TEST_DIR/file1.txt
assert_not_exists $TEST_DIR/file2.txt
assert_exists $TEST_DIR/folder1

cleanup
printf "%s[SUCCESS]%s All integration tests passed!\n" "$GREEN" "$NC"
