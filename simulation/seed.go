package simulation

import (
	"context"
	"fmt"

	"github.com/orange-cat-investments/oci/pkg/search/meilisearch"
)

// SeedSearchFixtures seeds development and simulation search documents into Meilisearch.
// Permitted under AGENTS.md Section 3 (Allowed Hardcoding Zones: Platform simulation driver).
func SeedSearchFixtures(ctx context.Context, meiliURL, meiliKey string) error {
	client := meilisearch.NewClient(meiliURL, meiliKey)

	_ = client.EnsureIndex(ctx, "customer_catalog", "id")
	_ = client.EnsureIndex(ctx, "employee_knowledge", "id")

	customerDocs := []map[string]interface{}{
		{"id": "fund-zoomie", "category": "fund", "symbol": "ZOOM", "title": "OCI Feline Behavioral Alpha Fund", "description": "Dynamic quantitative weighting based on active feline zoomie intervals."},
		{"id": "fund-catnip", "category": "fund", "symbol": "NIP", "title": "Organic Catnip Agriculture ESG Fund", "description": "Sustainable feline agricultural infrastructure and cat grass ventures."},
		{"id": "esg-welfare-2026", "category": "transparency", "title": "2026 Feline Workplace Welfare & Habitat Transparency", "description": "OSHA-compliant scratch post maintenance and continuous perch monitoring."},
		{"id": "stmt-tax-1099b", "category": "statement", "title": "Form 1099-B Proceeds From Broker and Barter Exchange", "description": "Annual consolidated tax statement for realized capital gains and DCA distributions."},
	}
	if err := client.IndexDocuments(ctx, "customer_catalog", customerDocs); err != nil {
		return fmt.Errorf("failed to seed customer_catalog: %w", err)
	}

	employeeDocs := []map[string]interface{}{
		{"id": "sop-laser-01", "category": "sop", "title": "SOP-014: Laser Pointer Telemetry & Biometrics", "snippet": "Standard clinical operating procedure for automated optical tracking and zoomie measurement."},
		{"id": "sop-nip-02", "category": "sop", "title": "SOP-089: Catnip Dispensary Quality Assurance", "snippet": "Guidelines for testing organic potency, THC-free feline compliance, and batch seals."},
		{"id": "vhr-garfield", "category": "vhr", "title": "VHR: Garfield Clinical Notes (Feline Alpha)", "snippet": "Dr. Sarah Chen: Weight 14.2 kg. Prescribed 15-minute scheduled cardboard box play sessions."},
		{"id": "vhr-chloe", "category": "vhr", "title": "VHR: Chloe Biometric & Activity Log", "snippet": "Active zoomies observed in Perch Zone Bravo. Resting heart rate nominal at 135 bpm."},
		{"id": "ticket-it-101", "category": "ticket", "title": "Switch Port Error on Edge Switch Habitat Alpha", "snippet": "High packet error rate on port ge-0/0/4 connected to high-speed feline camera 1."},
		{"id": "asset-cam-4k", "category": "asset", "title": "Edge 4K Habitat Camera (Firmware v2.1)", "snippet": "Sony IMX sensor edge vision node with real-time pose estimation inference."},
	}
	if err := client.IndexDocuments(ctx, "employee_knowledge", employeeDocs); err != nil {
		return fmt.Errorf("failed to seed employee_knowledge: %w", err)
	}

	return nil
}
