package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/orange-cat-investments/oci/simulation"
)

func main() {
	var (
		unifiedURL   = flag.String("url", "", "Unified base URL or staging domain (e.g. https://dev.api.oci.local, http://localhost:8080)")
		serverURL    = flag.String("server-url", "", "Base URL for OCI Domain API Server (overrides -url)")
		bffURL       = flag.String("bff-url", "", "Base URL for OCI Employee BFF (overrides -url)")
		custBFFURL   = flag.String("cust-bff-url", "", "Base URL for OCI Customer BFF (overrides -url)")
		forgejoURL   = flag.String("forgejo-url", "", "Base URL for Forgejo Git repository host (overrides -url)")
		brokerURL    = flag.String("broker-url", "", "Base URL for Mock Broker API (overrides -url)")
		bankURL      = flag.String("bank-url", "", "Base URL for Mock Bank API (overrides -url)")
		mode         = flag.String("mode", "live", "Simulation mode: 'live', 'step', 'continuous'")
		persona      = flag.String("persona", "all", "Target persona ID or Name (or 'all')")
		iterations   = flag.Int("iterations", 1, "Number of daily simulation loops to perform")
		delayMs      = flag.Int("delay", 100, "Pause delay between actions in milliseconds")
		verbose      = flag.Bool("verbose", true, "Print detailed logs for each action execution")
		listPersonas = flag.Bool("list-personas", false, "List all registered employee personas and exit")
		seedSearch   = flag.Bool("seed-search", false, "Seed Meilisearch indexes with development fixtures before running simulation")
		meiliURL     = flag.String("meili-url", "", "Meilisearch Base URL for search seeding (default from MEILISEARCH_URL or http://localhost:7700)")
	)

	flag.Parse()

	if *seedSearch {
		mURL := *meiliURL
		if mURL == "" {
			mURL = os.Getenv("MEILISEARCH_URL")
		}
		if mURL == "" {
			mURL = "http://localhost:7700"
		}
		mKey := os.Getenv("MEILI_MASTER_KEY")
		if err := simulation.SeedSearchFixtures(context.Background(), mURL, mKey); err != nil {
			log.Printf("Warning: failed to seed Meilisearch fixtures: %v", err)
		} else {
			log.Printf("Successfully seeded Meilisearch development fixtures to %s", mURL)
		}
	}

	if *listPersonas {
		registry := simulation.NewRegistry()
		fmt.Println("Registered Orange Cat Investments (OCI) Employee Personas:")
		fmt.Println("=========================================================")
		for _, p := range registry.List() {
			fmt.Printf("- %s (%s)\n  Type: %s | Title: %s\n  Department: %s | Actions: %d\n",
				p.Name, p.ID, p.Type, p.RoleTitle, p.Department, len(p.Actions))
		}
		os.Exit(0)
	}

	resolvedServer, resolvedBFF, resolvedCustBFF, resolvedForgejo := simulation.ResolveRoutingURLs(*unifiedURL, *serverURL, *bffURL, *custBFFURL, *forgejoURL)
	resolvedBroker, resolvedBank := simulation.ResolveExternalURLs(*unifiedURL, *brokerURL, *bankURL)

	cfg := &simulation.Config{
		ServerURL:      resolvedServer,
		EmployeeBFFURL: resolvedBFF,
		CustomerBFFURL: resolvedCustBFF,
		ForgejoURL:     resolvedForgejo,
		BrokerURL:      resolvedBroker,
		BankURL:        resolvedBank,
		Mode:           *mode,
		TargetPersona:  *persona,
		Iterations:     *iterations,
		ActionDelay:    time.Duration(*delayMs) * time.Millisecond,
		Verbose:        *verbose,
	}

	driver, err := simulation.NewDriver(cfg)
	if err != nil {
		log.Fatalf("Failed to create simulation driver: %v", err)
	}
	defer driver.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	summary, err := driver.Run(ctx)
	if err != nil && err != context.Canceled {
		log.Fatalf("Simulation execution error: %v", err)
	}

	fmt.Println("\n================ Simulation Execution Summary ================")
	fmt.Printf("Total Actions Executed : %d\n", summary.TotalActions)
	fmt.Printf("Successful Actions     : %d\n", summary.Successes)
	fmt.Printf("Failed Actions         : %d\n", summary.Failures)
	fmt.Printf("Total Elapsed Time     : %s\n", summary.Duration)
	fmt.Println("==============================================================")
}
