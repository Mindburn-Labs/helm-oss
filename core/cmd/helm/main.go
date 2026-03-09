package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/Mindburn-Labs/helm-oss/core/pkg/artifacts"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/auth"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/crypto"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/executor"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/guardian"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/identity"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/mcp"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/pack"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/prg"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/registry"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/store"
	"github.com/Mindburn-Labs/helm-oss/core/pkg/store/ledger"

	_ "github.com/lib/pq" // Postgres Driver
)

// Dispatcher
func main() {
	os.Exit(Run(os.Args, os.Stdout, os.Stderr))
}

// startServer is a variable to allow mocking in tests
var startServer = runServer

// Run is the entrypoint for testing
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) < 2 {
		// Default to server
		startServer()
		return 0
	}

	switch args[1] {
	case "onboard":
		return runOnboardCmd(args[2:], stdout, stderr)
	case "demo":
		return runDemoCmd(args[2:], stdout, stderr)
	case "sandbox":
		return runSandboxCmd(args[2:], stdout, stderr)
	case "mcp":
		return runMCPCmd(args[2:], stdout, stderr)
	case "proxy":
		return runProxyCmd(args[2:], stdout, stderr)
	case "export":
		return runExportCmd(args[2:], stdout, stderr)
	case "verify":
		return runVerifyCmd(args[2:], stdout, stderr)
	case "replay":
		return runReplayCmd(args[2:], stdout, stderr)
	case "conform", "conformance":
		return runConform(args[2:], stdout, stderr)
	case "freeze":
		return runFreezeCmd(args[2:], stdout, stderr, "freeze")
	case "unfreeze":
		return runFreezeCmd(args[2:], stdout, stderr, "unfreeze")
	case "risk-summary":
		return runRiskCmd(args[2:], stdout, stderr)
	case "doctor":
		return runDoctorCmd(stdout, stderr)
	case "init":
		return runInitCmd(args[2:], stdout, stderr)
	case "trust":
		if len(args) < 3 {
			_, _ = fmt.Fprintln(stderr, "Usage: helm trust <add-key|revoke-key|list-keys>")
			return 2
		}
		return runTrustCmd(args[2:], stdout, stderr)
	case "threat":
		if len(args) < 3 {
			_, _ = fmt.Fprintln(stderr, "Usage: helm threat <scan|test> [flags]")
			return 2
		}
		return runThreatCmd(args[2:], stdout, stderr)
	case "server", "serve":
		startServer()
		return 0
	case "health":
		return runHealthCmd(stdout, stderr)
	case "control-room":
		return runControlRoom(args[2:], stdout, stderr)
	case "coverage":
		handleCoverage(args[2:])
		return 0
	case "pack":
		return runPackCmd(args[2:], stdout, stderr)
	case "incident":
		return runIncidentCmd(args[2:], stdout, stderr)
	case "run":
		if len(args) > 2 && args[2] == "maintenance" {
			return runMaintenanceCmd(args[3:], stdout, stderr)
		}
		fmt.Fprintln(stderr, "Usage: helm run maintenance [--once|--schedule]")
		return 2
	case "brief":
		return runBriefCmd(args[2:], stdout, stderr)
	case "policy":
		return runPolicyCmd(args[2:], stdout, stderr)
	case "bundle":
		return runBundleCmd(args[2:], stdout, stderr)
	case "help", "--help", "-h":
		printUsage(stdout)
		return 0
	case "version", "--version", "-v":
		fmt.Fprintf(stdout, "%sHELM%s %s (%s)\n", ColorBold, ColorReset, displayVersion(), displayCommit())
		fmt.Fprintf(stdout, "  Report Schema:          %s\n", reportSchemaVersion)
		fmt.Fprintf(stdout, "  EvidencePack Schema:    1\n")
		fmt.Fprintf(stdout, "  Compatibility Schema:   1\n")
		fmt.Fprintf(stdout, "  MCP Bundle Schema:      1\n")
		fmt.Fprintf(stdout, "  Build Time:             %s\n", displayBuildTime())
		return 0
	default:
		if args[1][0] == '-' {
			startServer()
			return 0
		} else {
			_, _ = fmt.Fprintf(stderr, "Unknown command: %s\n", args[1])
			printUsage(stderr)
			return 2
		}
	}
}

// ANSI Colors
const (
	ColorReset  = "\033[0m"
	ColorBold   = "\033[1m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorGray   = "\033[37m"
)

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "")
	fmt.Fprintf(w, "%sHELM Kernel %s%s\n", ColorBold+ColorBlue, displayVersion(), ColorReset)
	fmt.Fprintf(w, "%sModels propose. The kernel disposes.%s\n", ColorGray, ColorReset)
	fmt.Fprintln(w, "")
	fmt.Fprintf(w, "%sUSAGE:%s\n", ColorBold, ColorReset)
	fmt.Fprintln(w, "  helm <command> [flags]")
	fmt.Fprintln(w, "")

	printSection(w, "QUICKSTART")
	printCommand(w, "onboard", "One-command local setup (SQLite + keys + config)")
	printCommand(w, "demo", "Run governed demonstrations (demo company)")

	printSection(w, "KERNEL")
	printCommand(w, "server", "Run the HELM server (default)")
	printCommand(w, "proxy", "OpenAI-compatible governance proxy")
	printCommand(w, "sandbox", "Governed sandbox execution (exec, conform)")
	printCommand(w, "doctor", "Check system health and configuration")
	printCommand(w, "health", "Check server health (HTTP)")
	printCommand(w, "init", "Initialize a new HELM project")

	printSection(w, "SKILL LIFECYCLE")
	printCommand(w, "pack propose", "Create a SkillCandidate artifact")
	printCommand(w, "pack build", "Build a Pack from a candidate")
	printCommand(w, "pack test", "Run conformance on a Pack")
	printCommand(w, "pack promote", "Promote a Pack (requires approval)")
	printCommand(w, "pack install", "Install a promoted Pack")
	printCommand(w, "pack list", "List candidates, packs, and installed skills")

	printSection(w, "MAINTENANCE")
	printCommand(w, "incident", "Manage incidents (list, show, ack, create)")
	printCommand(w, "run maintenance", "Run governed self-fixing maintenance")
	printCommand(w, "brief daily", "Generate daily system health brief")
	printCommand(w, "policy test", "Run policy fixtures (--dir policies)")
	printCommand(w, "policy init", "Generate starter policy (--template deny-first)")
	printCommand(w, "policy templates", "List available policy templates")

	printSection(w, "MCP DISTRIBUTION")
	printCommand(w, "mcp serve", "Start HELM MCP server (stdio or HTTP)")
	printCommand(w, "mcp install", "Install MCP for Claude Code")
	printCommand(w, "mcp pack", "Generate .mcpb for Claude Desktop")
	printCommand(w, "mcp print-config", "Print config for Windsurf/Codex/VS Code/Cursor")

	printSection(w, "CONFORMANCE & VERIFICATION")
	printCommand(w, "conform", "Run conformance gates (--profile, --json)")
	printCommand(w, "verify", "Verify EvidencePack bundle (--bundle, --json)")
	printCommand(w, "replay", "Replay and verify from tapes (--evidence)")
	printCommand(w, "export", "Export EvidencePack (--evidence, --out)")

	printSection(w, "OPERATIONS")
	printCommand(w, "freeze", "Activate global freeze (--principal)")
	printCommand(w, "unfreeze", "Deactivate freeze (--principal)")
	printCommand(w, "risk-summary", "Risk assessment (--effect, --list)")

	printSection(w, "TRUST MANAGEMENT")
	printCommand(w, "trust", "Manage trust root keys (add/revoke/list)")

	printSection(w, "THREAT DETECTION")
	printCommand(w, "threat scan", "Scan text input for threat signals")
	printCommand(w, "threat test", "Run adversarial threat scanner test suite")

	printSection(w, "POLICY BUNDLES")
	printCommand(w, "bundle list", "List loaded policy bundles")
	printCommand(w, "bundle verify", "Verify bundle integrity (--file, --hash)")
	printCommand(w, "bundle inspect", "Inspect bundle without activating")

	printSection(w, "UTILITIES")
	printCommand(w, "pack create", "Create a deterministic evidence pack (.tar)")
	printCommand(w, "pack verify", "Verify an evidence pack's integrity")
	printCommand(w, "coverage", "Show coverage statistics")
	printCommand(w, "help", "Show this help")
	fmt.Fprintln(w, "")
}

func printSection(w io.Writer, title string) {
	fmt.Fprintf(w, "%s%s:%s\n", ColorBold+ColorCyan, title, ColorReset)
}

func printCommand(w io.Writer, name, desc string) {
	fmt.Fprintf(w, "  %s%-12s%s %s\n", ColorGreen, name, ColorReset, desc)
}

func handleCoverage(args []string) {
	log.Println("[helm] coverage factory: ready")
}

func handlePackCreate(args []string) int {
	cmd := flag.NewFlagSet("pack create", flag.ContinueOnError)
	cmd.SetOutput(os.Stderr)

	var (
		sessionID   string
		receiptsDir string
		outPath     string
		jsonOutput  bool
	)

	cmd.StringVar(&sessionID, "session", "", "Session ID for the evidence pack (REQUIRED)")
	cmd.StringVar(&receiptsDir, "receipts", "", "Directory containing receipt files (REQUIRED)")
	cmd.StringVar(&outPath, "out", "", "Output path for the .tar pack (REQUIRED)")
	cmd.BoolVar(&jsonOutput, "json", false, "Output result as JSON")

	if err := cmd.Parse(args); err != nil {
		return 2
	}

	if sessionID == "" || receiptsDir == "" || outPath == "" {
		fmt.Fprintln(os.Stderr, "Error: --session, --receipts, and --out are required")
		cmd.Usage()
		return 2
	}

	// Read all files from receipts directory
	files := make(map[string][]byte)
	err := filepath.Walk(receiptsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		relPath, _ := filepath.Rel(receiptsDir, path)
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", relPath, err)
		}
		files[relPath] = data
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading receipts: %v\n", err)
		return 2
	}

	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "Error: no files found in receipts directory")
		return 2
	}

	// Auto-include proofgraph.json if it exists alongside receipts (Gap #22)
	pgPath := filepath.Join(receiptsDir, "proofgraph.json")
	if _, err := os.Stat(pgPath); err == nil {
		if _, exists := files["proofgraph.json"]; !exists {
			data, readErr := os.ReadFile(pgPath)
			if readErr == nil {
				files["proofgraph.json"] = data
			}
		}
	}

	// Auto-include trust_roots.json from artifacts if it exists (Gap #28)
	for _, trustPath := range []string{
		filepath.Join(receiptsDir, "..", "artifacts", "trust_roots.json"),
		filepath.Join(receiptsDir, "trust_roots.json"),
		"artifacts/trust_roots.json",
	} {
		if _, err := os.Stat(trustPath); err == nil {
			if _, exists := files["trust_roots.json"]; !exists {
				data, readErr := os.ReadFile(trustPath)
				if readErr == nil {
					files["trust_roots.json"] = data
				}
			}
			break
		}
	}

	// Create the pack
	if err := ExportPack(sessionID, files, outPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating pack: %v\n", err)
		return 2
	}

	if jsonOutput {
		result := map[string]any{
			"session_id": sessionID,
			"pack_path":  outPath,
			"file_count": len(files),
			"status":     "created",
		}
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
	} else {
		fmt.Printf("✅ Evidence pack created: %s (%d files)\n", outPath, len(files))
	}

	return 0
}

func handlePackVerify(args []string) int {
	cmd := flag.NewFlagSet("pack verify", flag.ContinueOnError)
	cmd.SetOutput(os.Stderr)

	var (
		bundlePath string
		jsonOutput bool
	)

	cmd.StringVar(&bundlePath, "bundle", "", "Path to evidence pack .tar (REQUIRED)")
	cmd.BoolVar(&jsonOutput, "json", false, "Output result as JSON")

	if err := cmd.Parse(args); err != nil {
		return 2
	}

	if bundlePath == "" {
		fmt.Fprintln(os.Stderr, "Error: --bundle is required")
		cmd.Usage()
		return 2
	}

	manifest, err := VerifyPack(bundlePath)
	if err != nil {
		if jsonOutput {
			result := map[string]any{
				"bundle": bundlePath,
				"valid":  false,
				"error":  err.Error(),
			}
			data, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(data))
		} else {
			fmt.Fprintf(os.Stderr, "❌ Verification failed: %v\n", err)
		}
		return 1
	}

	if jsonOutput {
		result := map[string]any{
			"bundle":      bundlePath,
			"valid":       true,
			"session_id":  manifest.SessionID,
			"version":     manifest.Version,
			"exported_at": manifest.ExportedAt,
			"file_count":  len(manifest.FileHashes),
		}
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
	} else {
		fmt.Printf("✅ Pack verified: %s\n", bundlePath)
		fmt.Printf("   Session:  %s\n", manifest.SessionID)
		fmt.Printf("   Version:  %s\n", manifest.Version)
		fmt.Printf("   Exported: %s\n", manifest.ExportedAt)
		fmt.Printf("   Files:    %d\n", len(manifest.FileHashes))
	}

	return 0
}

//nolint:gocognit,gocyclo
func runServer() {
	fmt.Fprintf(os.Stdout, "%sHELM Kernel starting...%s\n", ColorBold+ColorBlue, ColorReset)
	ctx := context.Background()
	logger := slog.Default()

	var (
		db           *sql.DB
		lgr          ledger.Ledger
		receiptStore store.ReceiptStore
		err          error
	)

	// 0.2 Connect to Database (Infrastructure)
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		fmt.Fprintf(os.Stdout, "ℹ️  DATABASE_URL not set. Falling back to %sLite Mode%s (SQLite).\n", ColorBold+ColorCyan, ColorReset)
		db, lgr, receiptStore, err = setupLiteMode(ctx)
		if err != nil {
			log.Fatalf("Failed to setup Lite Mode: %v", err)
		}
	} else {
		db, err = sql.Open("postgres", dbURL)
		if err != nil {
			log.Fatalf("Failed to connect to DB: %v", err)
		}
		if err := db.PingContext(ctx); err != nil {
			log.Fatalf("DB Ping failed: %v", err)
		}
		log.Println("[helm] postgres: connected")

		// Use Postgres Ledger
		pl := ledger.NewPostgresLedger(db)
		if err := pl.Init(ctx); err != nil {
			log.Fatalf("Failed to init ledger: %v", err)
		}
		lgr = pl
		ps := store.NewPostgresReceiptStore(db)
		if err := ps.Init(ctx); err != nil {
			log.Fatalf("Failed to init receipt store: %v", err)
		}
		receiptStore = ps
	}

	// 1. Initialize Kernel Layers
	// Identity KeySet (TCB: cryptographic key management for signing authority)
	keySet, err := identity.NewInMemoryKeySet()
	if err != nil {
		log.Fatalf("Failed to init KeySet: %v", err)
	}
	jwtValidator := auth.NewJWTValidator(keySet)

	// Signing Authority
	signer, err := loadOrGenerateSigner()
	if err != nil {
		log.Fatalf("Failed to init signer: %v", err)
	}
	verifier, _ := crypto.NewEd25519Verifier(signer.PublicKeyBytes())
	fmt.Fprintf(os.Stdout, "🔑 Trust Root: %s%s%s\n", ColorBold+ColorGreen, signer.PublicKey(), ColorReset)

	// 2. Registry
	reg := registry.NewPostgresRegistry(db)
	if err := reg.Init(ctx); err != nil {
		log.Fatalf("Failed to init registry: %v", err)
	}
	log.Println("[helm] registry: ready")

	// Pack Verifier
	regAdapter := pack.NewRegistryAdapter(reg)
	packVerifier := pack.NewVerifier(regAdapter)

	// Artifact Store
	artStore, _ := artifacts.NewFileStore("data/artifacts")
	artRegistry := artifacts.NewRegistry(artStore, verifier)

	// === SUBSYSTEM WIRING ===
	services, svcErr := NewServices(ctx, db, artStore, logger)
	if svcErr != nil {
		log.Printf("Services init (non-fatal, degraded mode): %v", svcErr)
	}

	// 2.5 PRG & Guardian
	ruleGraph := prg.NewGraph()
	// Bootstrap a minimal allow rule so governed chat inference is usable out of the box.
	// Production deployments should replace this with a loaded policy bundle.
	if err := ruleGraph.AddRule("LLM_INFERENCE", prg.RequirementSet{
		ID:    "bootstrap-llm-inference",
		Logic: prg.AND,
	}); err != nil {
		log.Fatalf("Failed to add bootstrap PRG rule: %v", err)
	}

	// Guardian
	guard := guardian.NewGuardian(signer, ruleGraph, artRegistry)

	// 3. Executor
	catalog := mcp.NewInMemoryCatalog()

	safeExec := executor.NewSafeExecutor(
		verifier,
		signer,
		nil, // driver
		receiptStore,
		artStore,
		nil, // outbox
		"sha256:production_verified_hash_v2",
		nil,      // audit
		nil,      // meter (removed enterprise metering)
		nil,      // outputSchemaRegistry (MVP: no pinned output schemas)
		time.Now, // Authority Clock
	)

	// Suppress unused variable warnings
	_ = lgr
	_ = catalog
	_ = safeExec
	_ = packVerifier
	_ = jwtValidator

	// Register Subsystem Routes
	var extraRoutes func(*http.ServeMux)
	if services != nil {
		services.Guardian = guard
		extraRoutes = func(mux *http.ServeMux) {
			RegisterSubsystemRoutes(mux, services)
		}
	}

	// Start API Server
	go func() {
		port := 8080
		mux := http.NewServeMux()

		// Register subsystem routes
		if extraRoutes != nil {
			extraRoutes(mux)
		}

		log.Printf("[helm] API server: :%d", port)
		//nolint:gosec
		if err := http.ListenAndServe(fmt.Sprintf(":%d", port), mux); err != nil {
			logger.Error("API server failed", "error", err)
			return
		}
	}()

	// Health Server
	healthMux := http.NewServeMux()
	healthHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}
	healthMux.HandleFunc("/health", healthHandler)
	healthMux.HandleFunc("/healthz", healthHandler)

	go func() {
		log.Printf("[helm] health server: :8081")
		//nolint:gosec // Intentionally listening on all interfaces
		if err := http.ListenAndServe(":8081", healthMux); err != nil {
			log.Printf("[helm] health server error: %v", err)
		}
	}()

	log.Println("[helm] ready: http://localhost:8080")
	log.Println("[helm] press ctrl+c to stop")

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("[helm] shutting down")
}

func runHealthCmd(out, errOut io.Writer) int {
	resp, err := http.Get("http://localhost:8081/healthz")
	if err != nil {
		fmt.Fprintf(errOut, "Health check failed: %v\n", err)
		return 1
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(errOut, "Health check failed: status %d\n", resp.StatusCode)
		return 1
	}

	fmt.Fprintln(out, "OK")
	return 0
}
