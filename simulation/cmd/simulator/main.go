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
		serverURL    = flag.String("server-url", "http://localhost:8080", "Base URL for OCI Domain API Server")
		bffURL       = flag.String("bff-url", "http://localhost:8081", "Base URL for OCI Employee BFF")
		mode         = flag.String("mode", "mock", "Simulation mode: 'mock', 'live', 'step', 'continuous'")
		persona      = flag.String("persona", "all", "Target persona ID or Name (or 'all')")
		iterations   = flag.Int("iterations", 1, "Number of daily simulation loops to perform")
		delayMs      = flag.Int("delay", 100, "Pause delay between actions in milliseconds")
		verbose      = flag.Bool("verbose", true, "Print detailed logs for each action execution")
		listPersonas = flag.Bool("list-personas", false, "List all registered employee personas and exit")
	)

	flag.Parse()

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

	cfg := &simulation.Config{
		ServerURL:      *serverURL,
		EmployeeBFFURL: *bffURL,
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
