// Package cmd provides the command-line interface for nuke
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"nuke/internal/config"
	"nuke/internal/deleter"
	"nuke/internal/filter"
	"nuke/internal/scanner"
	"nuke/internal/trash"
	"nuke/internal/utils"

	"github.com/schollz/progressbar/v3"
)

// CLI flags and options
var (
	dryRun       bool
	recursive    bool
	force        bool
	interactive  bool
	shred        bool
	verbose      bool
	emptyTrash   bool
	cleanupTrash bool
	restoreFile  string
	showTrash    bool
	olderThan    string
	newerThan    string
	sizeFilter   string
	exclude      []string
	include      []string
	regexPattern string
	noCountdown  bool
	workers      int
)

// Execute runs the main CLI logic
func Execute() error {
	args := os.Args[1:]

	// Parse flags and get targets
	targets, err := parseArgs(args)
	if err != nil {
		return err
	}

	// Handle special commands
	if emptyTrash {
		return handleEmptyTrash()
	}

	if cleanupTrash {
		return handleCleanupTrash(config.LoadConfig())
	}

	if showTrash {
		return handleShowTrash()
	}

	if restoreFile != "" {
		return handleRestore(restoreFile)
	}

	// Validate targets
	if len(targets) == 0 {
		printHelp()
		return nil
	}

	// Load protected paths configuration
	cfg := config.LoadConfig()

	// Create filter options
	filterOpts, err := createFilterOptions()
	if err != nil {
		return fmt.Errorf("invalid filter options: %w", err)
	}

	// Scan targets and collect files
	fmt.Println("🔍 Scanning targets...")
	files, err := scanTargets(targets, filterOpts, cfg)
	if err != nil {
		return fmt.Errorf("scan error: %w", err)
	}

	if len(files) == 0 {
		fmt.Println("✅ No files match the specified criteria.")
		return nil
	}

	// Calculate total size
	totalSize := calculateTotalSize(files)

	// Display summary
	displaySummary(files, totalSize)

	// Check for dangerous patterns
	if err := checkDangerousPatterns(targets, files); err != nil {
		return err
	}

	// Dry run mode - just show what would be deleted
	if dryRun {
		fmt.Println("\n📋 DRY RUN - The following would be deleted:")
		for _, f := range files {
			fmt.Printf("  %s (%s)\n", f.Path, utils.FormatSize(f.Size))
		}
		fmt.Println("\n✅ Dry run complete. No files were modified.")
		return nil
	}

	// Interactive mode - ask for each file
	if interactive {
		return handleInteractiveDelete(files, cfg)
	}

	// Standard confirmation
	if !force {
		if !confirmDeletion(len(files)) {
			fmt.Println("❌ Operation cancelled.")
			return nil
		}
	}

	// Countdown timer for operations with more than 5 files
	if len(files) > 5 && !noCountdown {
		if !countdownWithAbort(5) {
			fmt.Println("\n❌ Operation aborted.")
			return nil
		}
	}

	// Perform deletion
	return performDeletion(files, cfg)
}

// parseArgs parses command line arguments and returns targets
func parseArgs(args []string) ([]string, error) {
	var targets []string

	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch {
		case arg == "-h" || arg == "--help":
			printHelp()
			os.Exit(0)
		case arg == "--dry-run":
			dryRun = true
		case arg == "-r" || arg == "--recursive":
			recursive = true
		case arg == "-f" || arg == "--force":
			force = true
		case arg == "-i" || arg == "--interactive":
			interactive = true
		case arg == "--shred":
			shred = true
		case arg == "-v" || arg == "--verbose":
			verbose = true
		case arg == "--empty-trash":
			emptyTrash = true
		case arg == "--cleanup-trash":
			cleanupTrash = true
		case arg == "--show-trash":
			showTrash = true
		case arg == "--no-countdown":
			noCountdown = true
		case strings.HasPrefix(arg, "--restore="):
			restoreFile = strings.TrimPrefix(arg, "--restore=")
		case strings.HasPrefix(arg, "--older-than="):
			olderThan = strings.TrimPrefix(arg, "--older-than=")
		case strings.HasPrefix(arg, "--newer-than="):
			newerThan = strings.TrimPrefix(arg, "--newer-than=")
		case strings.HasPrefix(arg, "--size="):
			sizeFilter = strings.TrimPrefix(arg, "--size=")
		case strings.HasPrefix(arg, "--exclude="):
			exclude = append(exclude, strings.TrimPrefix(arg, "--exclude="))
		case strings.HasPrefix(arg, "--include="):
			include = append(include, strings.TrimPrefix(arg, "--include="))
		case strings.HasPrefix(arg, "--regex="):
			regexPattern = strings.TrimPrefix(arg, "--regex=")
		case strings.HasPrefix(arg, "--workers="):
			fmt.Sscanf(strings.TrimPrefix(arg, "--workers="), "%d", &workers)
		case strings.HasPrefix(arg, "-"):
			return nil, fmt.Errorf("unknown option: %s", arg)
		default:
			targets = append(targets, arg)
		}
	}

	// Set default workers
	if workers <= 0 {
		workers = 8
	}

	return targets, nil
}

// createFilterOptions creates filter options from CLI flags
func createFilterOptions() (*filter.Options, error) {
	opts := &filter.Options{
		Exclude: exclude,
		Include: include,
	}

	// Parse older-than filter
	if olderThan != "" {
		duration, err := utils.ParseDuration(olderThan)
		if err != nil {
			return nil, fmt.Errorf("invalid --older-than value: %w", err)
		}
		cutoff := time.Now().Add(-duration)
		opts.OlderThan = &cutoff
	}

	// Parse newer-than filter
	if newerThan != "" {
		duration, err := utils.ParseDuration(newerThan)
		if err != nil {
			return nil, fmt.Errorf("invalid --newer-than value: %w", err)
		}
		cutoff := time.Now().Add(-duration)
		opts.NewerThan = &cutoff
	}

	// Parse size filter
	if sizeFilter != "" {
		size, op, err := utils.ParseSizeFilter(sizeFilter)
		if err != nil {
			return nil, fmt.Errorf("invalid --size value: %w", err)
		}
		opts.SizeFilter = size
		opts.SizeOp = op
	}

	// Compile regex pattern
	if regexPattern != "" {
		re, err := regexp.Compile(regexPattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern: %w", err)
		}
		opts.Regex = re
	}

	return opts, nil
}

// scanTargets scans all targets and returns matching files
func scanTargets(targets []string, filterOpts *filter.Options, cfg *config.Config) ([]scanner.FileInfo, error) {
	var allFiles []scanner.FileInfo

	for _, target := range targets {
		// Expand glob patterns
		matches, err := filepath.Glob(target)
		if err != nil {
			return nil, fmt.Errorf("invalid pattern %s: %w", target, err)
		}

		if len(matches) == 0 {
			// Try as literal path
			matches = []string{target}
		}

		for _, match := range matches {
			absPath, err := filepath.Abs(match)
			if err != nil {
				continue
			}

			// Check if path is protected
			if cfg.IsProtected(absPath) {
				fmt.Printf("⚠️  Skipping protected path: %s\n", absPath)
				continue
			}

			files, err := scanner.Scan(absPath, recursive, filterOpts)
			if err != nil {
				if verbose {
					fmt.Printf("⚠️  Warning: %v\n", err)
				}
				continue
			}

			allFiles = append(allFiles, files...)
		}
	}

	return allFiles, nil
}

// calculateTotalSize calculates the total size of all files
func calculateTotalSize(files []scanner.FileInfo) int64 {
	var total int64
	for _, f := range files {
		total += f.Size
	}
	return total
}

// displaySummary shows a summary of what will be deleted
func displaySummary(files []scanner.FileInfo, totalSize int64) {
	fmt.Printf("\n📊 Summary:\n")
	fmt.Printf("   Files to delete: %d\n", len(files))
	fmt.Printf("   Total size: %s\n", utils.FormatSize(totalSize))

	if verbose {
		fmt.Println("\n📁 Files:")
		for _, f := range files {
			fmt.Printf("   %s (%s)\n", f.Path, utils.FormatSize(f.Size))
		}
	}
}

// checkDangerousPatterns checks for dangerous deletion patterns
func checkDangerousPatterns(targets []string, files []scanner.FileInfo) error {
	dangerous := false
	var reason string

	for _, target := range targets {
		absPath, _ := filepath.Abs(target)

		// Check for root directory
		if absPath == "/" {
			dangerous = true
			reason = "Attempting to delete root directory (/)"
			break
		}

		// Check for wildcard-only patterns
		if target == "*" || target == "/*" || target == "./*" {
			dangerous = true
			reason = "Wildcard-only pattern detected"
			break
		}

		// Check for system directories
		systemDirs := []string{"/bin", "/sbin", "/usr", "/etc", "/var", "/lib", "/boot"}
		for _, dir := range systemDirs {
			if absPath == dir || strings.HasPrefix(absPath, dir+"/") {
				if len(files) > 100 {
					dangerous = true
					reason = fmt.Sprintf("Attempting to delete many files in system directory: %s", dir)
					break
				}
			}
		}
	}

	if dangerous {
		fmt.Printf("\n⚠️  DANGEROUS OPERATION DETECTED!\n")
		fmt.Printf("   Reason: %s\n", reason)
		fmt.Printf("\n   To proceed, type 'yes I am sure': ")

		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input != "yes I am sure" {
			return fmt.Errorf("dangerous operation not confirmed")
		}
	}

	return nil
}

// confirmDeletion asks for y/n confirmation
func confirmDeletion(fileCount int) bool {
	fmt.Printf("\n❓ Proceed with deletion? [y/N]: ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	return input == "y" || input == "yes"
}

// countdownWithAbort shows a countdown timer that can be aborted with Ctrl+C
func countdownWithAbort(seconds int) bool {
	fmt.Printf("\n⏱️  Starting in %d seconds (Ctrl+C to abort)...\n", seconds)

	// Set up signal handler
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	done := make(chan bool, 1)

	go func() {
		for i := seconds; i > 0; i-- {
			select {
			case <-sigChan:
				done <- false
				return
			default:
				fmt.Printf("\r   %d...", i)
				time.Sleep(time.Second)
			}
		}
		fmt.Printf("\r       \n")
		done <- true
	}()

	select {
	case result := <-done:
		return result
	case <-sigChan:
		return false
	}
}

// handleInteractiveDelete handles interactive deletion mode
func handleInteractiveDelete(files []scanner.FileInfo, cfg *config.Config) error {
	reader := bufio.NewReader(os.Stdin)
	deleteAll := false

	var toDelete []scanner.FileInfo

	for _, f := range files {
		if deleteAll {
			toDelete = append(toDelete, f)
			continue
		}

		fmt.Printf("\n📄 %s (%s)\n", f.Path, utils.FormatSize(f.Size))
		fmt.Print("   Delete? [y/n/a/q] (yes/no/all/quit): ")

		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))

		switch input {
		case "y", "yes":
			toDelete = append(toDelete, f)
		case "a", "all":
			deleteAll = true
			toDelete = append(toDelete, f)
		case "q", "quit":
			if len(toDelete) > 0 {
				fmt.Printf("\n⚠️  %d files were marked for deletion before quit.\n", len(toDelete))
				fmt.Print("   Delete marked files? [y/N]: ")
				confirm, _ := reader.ReadString('\n')
				confirm = strings.TrimSpace(strings.ToLower(confirm))
				if confirm == "y" || confirm == "yes" {
					return performDeletion(toDelete, cfg)
				}
			}
			fmt.Println("❌ Operation cancelled.")
			return nil
		}
	}

	if len(toDelete) == 0 {
		fmt.Println("✅ No files selected for deletion.")
		return nil
	}

	return performDeletion(toDelete, cfg)
}

// performDeletion performs the actual deletion operation
func performDeletion(files []scanner.FileInfo, cfg *config.Config) error {
	fmt.Printf("\n🗑️  Deleting %d files...\n", len(files))

	// Create progress bar
	bar := progressbar.NewOptions(len(files),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowBytes(false),
		progressbar.OptionSetWidth(40),
		progressbar.OptionSetDescription("[cyan]Deleting[reset]"),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
	)

	// Create trash manager
	trashMgr, err := trash.NewManager()
	if err != nil {
		return fmt.Errorf("failed to initialize trash: %w", err)
	}

	// Create deleter
	del := deleter.New(workers, shred, trashMgr)

	// Track errors
	var errMu sync.Mutex
	var errors []error

	// Progress callback
	onProgress := func(path string, err error) {
		bar.Add(1)
		if err != nil {
			errMu.Lock()
			errors = append(errors, fmt.Errorf("%s: %w", path, err))
			errMu.Unlock()
			if verbose {
				fmt.Printf("\n⚠️  Error: %s: %v\n", path, err)
			}
		}
	}

	// Perform deletion
	del.Delete(files, onProgress)

	fmt.Println()

	// Report results
	successCount := len(files) - len(errors)
	fmt.Printf("\n✅ Successfully processed: %d files\n", successCount)

	if len(errors) > 0 {
		fmt.Printf("⚠️  Errors: %d\n", len(errors))
		if verbose {
			for _, e := range errors {
				fmt.Printf("   - %v\n", e)
			}
		}
	}

	if !shred {
		fmt.Println("\n💡 Files moved to trash. Use --empty-trash to permanently delete.")
		fmt.Printf("   Use --restore=<filename> to restore a file.\n")
	}

	return nil
}

// handleEmptyTrash empties the trash directory
func handleEmptyTrash() error {
	trashMgr, err := trash.NewManager()
	if err != nil {
		return err
	}

	items, size, err := trashMgr.List()
	if err != nil {
		return err
	}

	if len(items) == 0 {
		fmt.Println("🗑️  Trash is already empty.")
		return nil
	}

	fmt.Printf("🗑️  Trash contains %d items (%s)\n", len(items), utils.FormatSize(size))
	fmt.Print("   Empty trash permanently? [y/N]: ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	if input != "y" && input != "yes" {
		fmt.Println("❌ Operation cancelled.")
		return nil
	}

	if err := trashMgr.Empty(); err != nil {
		return err
	}

	fmt.Println("✅ Trash emptied successfully.")
	return nil
}

// handleRestore restores a file from trash
func handleRestore(filename string) error {
	trashMgr, err := trash.NewManager()
	if err != nil {
		return err
	}

	if err := trashMgr.Restore(filename); err != nil {
		return err
	}

	fmt.Printf("✅ Restored: %s\n", filename)
	return nil
}

// handleShowTrash shows what's in the trash
func handleShowTrash() error {
	trashMgr, err := trash.NewManager()
	if err != nil {
		return err
	}

	items, totalSize, err := trashMgr.List()
	if err != nil {
		return err
	}

	if len(items) == 0 {
		fmt.Println("🗑️  Trash is empty.")
		return nil
	}

	fmt.Printf("🗑️  Trash contents (%d items, %s):\n\n", len(items), utils.FormatSize(totalSize))

	for i, item := range items {
		daysAgo := int(time.Since(item.DeletedAt).Hours() / 24)
		fmt.Printf("%d. %s\n", i+1, filepath.Base(item.OriginalPath))
		fmt.Printf("   Original: %s\n", item.OriginalPath)
		fmt.Printf("   Size: %s\n", utils.FormatSize(item.Size))
		fmt.Printf("   Deleted: %d days ago (%s)\n", daysAgo, item.DeletedAt.Format("2006-01-02 15:04:05"))
		fmt.Println()
	}

	return nil
}

// handleCleanupTrash removes old files from trash based on retention policy
func handleCleanupTrash(cfg *config.Config) error {
	trashMgr, err := trash.NewManager()
	if err != nil {
		return err
	}

	fmt.Println("🧹 Running trash cleanup...")
	fmt.Printf("   Retention: %d days\n", cfg.TrashRetentionDays)
	fmt.Printf("   Max size: %d MB\n", cfg.TrashMaxSizeMB)

	itemsRemoved, bytesFreed, err := trashMgr.AutoCleanup(cfg.TrashRetentionDays, cfg.TrashMaxSizeMB)
	if err != nil {
		return err
	}

	if itemsRemoved == 0 {
		fmt.Println("✅ Trash is within limits. No cleanup needed.")
		return nil
	}

	fmt.Printf("\n✅ Cleanup complete:\n")
	fmt.Printf("   Items removed: %d\n", itemsRemoved)
	fmt.Printf("   Space freed: %s\n", utils.FormatSize(bytesFreed))

	return nil
}

// printHelp displays help information
func printHelp() {
	help := `
nuke - A safer, faster, and more user-friendly alternative to rm

USAGE:
    nuke [OPTIONS] <targets>...

DESCRIPTION:
    nuke is a command-line utility for deleting files safely. It provides
    soft-delete (trash) functionality, concurrent deletion, safety checks,
    and various filtering options.

OPTIONS:
    -h, --help           Show this help message
    -r, --recursive      Delete directories recursively
    -f, --force          Skip confirmation prompts
    -i, --interactive    Ask for confirmation for each file
    -v, --verbose        Show detailed output
    --dry-run            Show what would be deleted without actually deleting
    --shred              Securely overwrite files before deletion
    --no-countdown       Skip the countdown timer

TRASH OPERATIONS:
    --empty-trash        Permanently delete all files in trash
    --cleanup-trash      Auto-clean trash based on retention policy
    --show-trash         Show what's in the trash
    --restore=<file>     Restore a file from trash

FILTERING OPTIONS:
    --older-than=<dur>   Delete files older than duration (e.g., 30d, 24h)
    --newer-than=<dur>   Delete files newer than duration
    --size=<size>        Filter by size (+100M for >100MB, -1G for <1GB)
    --exclude=<pattern>  Exclude files matching glob pattern
    --include=<pattern>  Include only files matching glob pattern
    --regex=<pattern>    Match files using regex pattern

PERFORMANCE OPTIONS:
    --workers=<n>        Number of concurrent workers (default: 8)

EXAMPLES:
    nuke file.txt                    Delete a single file
    nuke -r directory/               Delete directory recursively
    nuke --dry-run *.log             Preview deletion of log files
    nuke --older-than=30d logs/      Delete logs older than 30 days
    nuke --size=+100M downloads/     Delete files larger than 100MB
    nuke --exclude=*.cfg config/     Delete all except .cfg files
    nuke --shred secret.txt          Securely delete sensitive file
    nuke --show-trash                Show what's in the trash
    nuke --cleanup-trash             Auto-cleanup old trash files
    nuke --empty-trash               Empty the trash permanently
    nuke --restore=file.txt          Restore file from trash

SAFETY FEATURES:
    - Protected paths: Certain system paths are protected from deletion
    - Dangerous pattern detection: Warns about risky operations like '/*'
    - Confirmation required: Asks before deleting
    - Countdown timer: 5-second countdown for large operations (Ctrl+C to abort)
    - Soft delete: Files are moved to trash by default (use --shred to bypass)

PROTECTED PATHS:
    The following paths are protected by default:
    /, /bin, /sbin, /usr, /etc, /var, /lib, /boot, /sys, /proc, /dev
    ~/.ssh, ~/.gnupg, .git/

    Additional paths can be configured in ~/.config/nuke/config.yaml
`
	fmt.Println(help)
}
