-- Orange Cat Investments (OCI) - Development & Simulation Seed Data Script
-- TARGET ENVIRONMENT: oci-dev ONLY (STRICTLY FORBIDDEN FOR PRODUCTION oci-core)
--
-- This script populates demo habitat zones, edge hardware devices, firmware releases,
-- the corporate feline and human workforce roster, onboarding checklists, customer
-- portfolios, tax lot sales, review cycles, IT tickets, and executive reports.

--------------------------------------------------------------------------------
-- Primary Habitat Location Zone
--------------------------------------------------------------------------------
INSERT INTO facilities.location_zones (zone_id, name, building, floor_level, description)
VALUES 
    ('018f3a9a-1111-7000-8000-000000000001', 'Alpha Sunbeam Lounge', 'HQ East', 2, 'Primary high-altitude perch lounge and observation zone')
ON CONFLICT (name) DO NOTHING;

--------------------------------------------------------------------------------
-- Initial Hardware Assets
--------------------------------------------------------------------------------
INSERT INTO facilities.hardware_assets (asset_id, serial_number, asset_type, model, status, zone_id)
VALUES 
    ('018e0000-0000-7000-8000-000000000001', 'QR-CAM-ORANGE-01', 'edge_camera', 'FelineCam-Pro-X', 'active', '018f3a9a-1111-7000-8000-000000000001'),
    ('018f3a9a-3333-7000-8000-000000000003', 'CAM-RIG-2025-01', 'edge_camera', '4K-Feline-Scope-V1', 'active', '018f3a9a-1111-7000-8000-000000000001'),
    ('018f3a9a-5555-7000-8000-000000000005', 'SW-CORE-01', 'networking_switch', 'Cisco-Catalyst-SMB-24P', 'active', '018f3a9a-1111-7000-8000-000000000001'),
    ('018f3a9a-6666-7000-8000-000000000006', 'GW-HABITAT-01', 'gateway', 'Edge-Gateway-IoT-V1', 'active', '018f3a9a-1111-7000-8000-000000000001')
ON CONFLICT (serial_number) DO NOTHING;

--------------------------------------------------------------------------------
-- Habitat Camera Streams (WebRTC / RTSP Ingestion)
--------------------------------------------------------------------------------
INSERT INTO core_invest.camera_streams (stream_id, camera_asset_id, stream_url, protocol, status, ingest_settings)
VALUES
    (
        '018f3a9a-c001-7000-8000-000000000001',
        '018e0000-0000-7000-8000-000000000001',
        'webrtc://streams.oci.local/live/habitat-cam-01',
        'webrtc',
        'active',
        '{"name": "Perch Alpha 4K WebRTC Live Feed", "zone": "Zone Alpha", "hls_url": "https://streams.oci.local/live/habitat-cam-01/index.m3u8"}'::jsonb
    ),
    (
        '018f3a9a-c002-7000-8000-000000000002',
        '018f3a9a-3333-7000-8000-000000000003',
        'webrtc://streams.oci.local/live/habitat-cam-02',
        'webrtc',
        'active',
        '{"name": "Nap Pod Beta Wide-Angle Feed", "zone": "Zone Beta", "hls_url": "https://streams.oci.local/live/habitat-cam-02/index.m3u8"}'::jsonb
    )
ON CONFLICT (stream_id) DO NOTHING;

--------------------------------------------------------------------------------
-- Firmware Release Catalog
--------------------------------------------------------------------------------
INSERT INTO facilities.firmware_releases (release_id, version, device_type, checksum, file_url, status)
VALUES 
    ('018e0000-0000-7000-8000-000000000002', 'v2.1.0-catos', 'edge_camera', 'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855', 'https://firmware.oci.local/v2.1.0.bin', 'published')
ON CONFLICT (device_type, version) DO NOTHING;

--------------------------------------------------------------------------------
-- Seed Workforce Employees
--------------------------------------------------------------------------------
INSERT INTO workforce.employees (employee_id, employee_type, first_name, last_name, email, role_title, department, status)
VALUES
    ('018f3a9a-e001-7000-8000-000000000001', 'feline', 'Garfield', NULL, NULL, 'Chief Observation Officer', 'Executive Feline Suite', 'active'),
    ('018f3a9a-e002-7000-8000-000000000002', 'feline', 'Barneby', NULL, NULL, 'Senior Alpha Perch Analyst', 'Alpha Perch Research', 'active'),
    ('018f3a9a-e003-7000-8000-000000000003', 'human', 'Alice', 'Vance', 'alice.vance@oci.local', 'Head of Human & Feline Resources', 'Workforce Operations', 'active'),
    ('018f3a9a-e004-7000-8000-000000000004', 'human', 'Elena', 'Rostova', 'elena.rostova@oci.local', 'Chief Veterinary Officer', 'Feline Health & Welfare', 'active'),
    ('018f3a9a-e005-7000-8000-000000000005', 'human', 'Rick', 'Newhire', 'rick.newhire@oci.local', 'Junior Operations Associate', 'Workforce Operations', 'onboarding'),
    ('018f3a9a-e006-7000-8000-000000000006', 'human', 'Elise', 'Dev', 'elise.dev@oci.local', 'Software Developer', 'Engineering & Platform Development', 'active'),
    ('018f3a9a-e007-7000-8000-000000000007', 'human', 'Frank', 'Operations', 'frank.ops@oci.local', 'Platform Security & K8s Infrastructure Lead', 'Platform Security & Infrastructure', 'active'),
    ('018f3a9a-e008-7000-8000-000000000008', 'human', 'Bob', 'Builder', 'bob.builder@oci.local', 'Lead Facilities & Edge Telemetry Engineer', 'Habitat Facilities & Infrastructure', 'active')
ON CONFLICT (employee_id) DO NOTHING;

--------------------------------------------------------------------------------
-- Employee Equipment & On-Call Alert Wearables (Pebble Watches & Laptops)
--------------------------------------------------------------------------------
INSERT INTO facilities.hardware_assets (asset_id, serial_number, asset_type, model, status, zone_id, assigned_employee_id)
VALUES
    ('018f3a9a-7777-7000-8000-000000000007', 'PEBBLE-FRANK-01', 'pebble_watch', 'Pebble Time Steel (Black)', 'active', '018f3a9a-1111-7000-8000-000000000001', '018f3a9a-e007-7000-8000-000000000007'),
    ('018f3a9a-7777-7000-8000-000000000008', 'PEBBLE-BOB-01', 'pebble_watch', 'Pebble Time (Red)', 'active', '018f3a9a-1111-7000-8000-000000000001', '018f3a9a-e008-7000-8000-000000000008'),
    ('018f3a9a-8888-7000-8000-000000000001', 'MBP-RICK-M3-01', 'laptop', 'MacBook Pro 16-inch M3 Max', 'active', '018f3a9a-1111-7000-8000-000000000001', '018f3a9a-e005-7000-8000-000000000005')
ON CONFLICT (serial_number) DO NOTHING;

--------------------------------------------------------------------------------
-- Seed Initial Onboarding Checklists
--------------------------------------------------------------------------------
INSERT INTO workforce.onboarding_checklists (checklist_id, employee_id, task_name, category, is_completed, assigned_to)
VALUES
    ('018f3a9a-c001-7000-8000-000000000001', '018f3a9a-e005-7000-8000-000000000005', 'Forgejo Account Provisioning & SPIFFE ID Issuance', 'access', false, '018f3a9a-e003-7000-8000-000000000003'),
    ('018f3a9a-c002-7000-8000-000000000002', '018f3a9a-e005-7000-8000-000000000005', 'Catnip Safety Orientation & Workstation Setup', 'training', false, '018f3a9a-e003-7000-8000-000000000003')
ON CONFLICT (checklist_id) DO NOTHING;

--------------------------------------------------------------------------------
-- Core Investment Portfolios for Retail Customers
--------------------------------------------------------------------------------
INSERT INTO core_invest.investment_portfolios (portfolio_id, customer_id, account_name, total_balance_usd, cash_balance_usd, status)
VALUES 
    ('018f3a9a-2222-7000-8000-000000000002', '018e0000-0000-7000-8000-000000000099', 'OCI Alpha Growth Fund', 128450.75, 32100.25, 'active'),
    ('018f3a9a-2222-7000-8000-000000000003', '018e0000-0000-7000-8000-000000000098', 'Chloe Spark High-Velocity Alpha', 85200.00, 15400.00, 'active')
ON CONFLICT (portfolio_id) DO NOTHING;

--------------------------------------------------------------------------------
-- Seed Initial Portfolio Holdings
--------------------------------------------------------------------------------
INSERT INTO core_invest.portfolio_holdings (
    holding_id, portfolio_id, symbol, quantity, cost_basis_usd, avg_price_usd, current_price_usd, market_value_usd, unrealized_gain_usd, unrealized_gain_pct
) VALUES
    ('018f3a9a-9999-7000-8000-000000000001', '018f3a9a-2222-7000-8000-000000000002', 'AAPL', 120.0000, 21420.00, 178.50, 224.30, 26916.00, 5496.00, 25.6583),
    ('018f3a9a-9999-7000-8000-000000000002', '018f3a9a-2222-7000-8000-000000000002', 'NVDA', 85.0000, 9367.00, 110.20, 138.80, 11798.00, 2431.00, 25.9528),
    ('018f3a9a-9999-7000-8000-000000000003', '018f3a9a-2222-7000-8000-000000000002', 'MSFT', 90.0000, 34200.00, 380.00, 448.20, 40338.00, 6138.00, 17.9474),
    ('018f3a9a-9999-7000-8000-000000000004', '018f3a9a-2222-7000-8000-000000000002', 'TSLA', 70.0000, 13300.00, 190.00, 247.12, 17298.40, 3998.40, 30.0632),
    ('018f3a9a-9999-7000-8000-000000000005', '018f3a9a-2222-7000-8000-000000000003', 'NVDA', 150.0000, 16860.00, 112.40, 138.80, 20820.00, 3960.00, 23.4875),
    ('018f3a9a-9999-7000-8000-000000000006', '018f3a9a-2222-7000-8000-000000000003', 'TSLA', 120.0000, 22260.00, 185.50, 247.12, 29654.40, 7394.40, 33.2183),
    ('018f3a9a-9999-7000-8000-000000000007', '018f3a9a-2222-7000-8000-000000000003', 'AAPL', 85.0000, 15308.50, 180.10, 224.30, 19065.50, 3757.00, 24.5420)
ON CONFLICT (portfolio_id, symbol) DO NOTHING;

--------------------------------------------------------------------------------
-- Seed Initial Tax Lot Sales for Form 1099-B Tax Documentation
--------------------------------------------------------------------------------
INSERT INTO core_invest.tax_lot_sales (
    lot_id, portfolio_id, symbol, description, quantity, date_acquired, date_sold,
    gross_proceeds_usd, cost_basis_usd, wash_sale_loss_disallowed, realized_gain_loss_usd, is_short_term, tax_year
) VALUES
    ('018f3a9a-aaaa-7000-8000-000000000001', '018f3a9a-2222-7000-8000-000000000002', 'NVDA', '50 shs NVDA', 50.0000, '2025-02-14', '2025-08-20', 6940.00, 5500.00, 0.00, 1440.00, true, 2025),
    ('018f3a9a-aaaa-7000-8000-000000000002', '018f3a9a-2222-7000-8000-000000000002', 'AAPL', '35 shs AAPL', 35.0000, '2025-01-10', '2025-09-15', 7850.00, 6200.00, 0.00, 1650.00, true, 2025),
    ('018f3a9a-aaaa-7000-8000-000000000003', '018f3a9a-2222-7000-8000-000000000002', 'MSFT', '20 shs MSFT', 20.0000, '2024-03-01', '2025-11-12', 8960.00, 7100.00, 0.00, 1860.00, false, 2025),
    ('018f3a9a-aaaa-7000-8000-000000000004', '018f3a9a-2222-7000-8000-000000000003', 'TSLA', '40 shs TSLA', 40.0000, '2025-03-15', '2025-07-22', 9884.80, 7420.00, 0.00, 2464.80, true, 2025),
    ('018f3a9a-aaaa-7000-8000-000000000005', '018f3a9a-2222-7000-8000-000000000003', 'NVDA', '60 shs NVDA', 60.0000, '2025-02-01', '2025-08-10', 8328.00, 6744.00, 0.00, 1584.00, true, 2025)
ON CONFLICT (lot_id) DO NOTHING;

--------------------------------------------------------------------------------
-- Automated Allocation Strategy
--------------------------------------------------------------------------------
INSERT INTO core_invest.allocation_strategies (strategy_id, name, trigger_activity, action_type, target_asset_symbol, multiplier, is_active)
VALUES 
    ('018f3a9a-4444-7000-8000-000000000004', 'Garfield Zoomies Multiplier', 'zooming', 'buy', 'AAPL', 1.5000, true)
ON CONFLICT (name) DO NOTHING;

--------------------------------------------------------------------------------
-- Initial IT Helpdesk & Customer Support Tickets
--------------------------------------------------------------------------------
INSERT INTO ops.it_tickets (
    ticket_id, forgejo_issue_id, forgejo_repo, title, body, state, severity, category, source, customer_id, assignee_username, author_username, is_urgent
) VALUES 
    (
        '018e0000-0000-7000-8000-000000000003', 101, 'oci-internal/helpdesk',
        'High Memory Usage Alert on Edge Gateway',
        'Edge camera gateway node memory exceeded 85% threshold in Habitat Alpha.',
        'open', 'urgent', 'hardware', 'internal', NULL, 'emp-human-frank', 'emp-human-frank', true
    ),
    (
        '018e0000-0000-7000-8000-000000000004', 102, 'oci-internal/helpdesk',
        'Sunbeam Perch Thermostat Calibration Drift',
        'Perch thermal sensor readings fluctuating +/- 4C outside comfort envelope.',
        'acknowledged', 'high', 'habitat', 'internal', NULL, 'emp-human-bob', 'emp-human-bob', false
    ),
    (
        '018e0000-0000-7000-8000-000000000005', 103, 'oci-internal/customer-support',
        'Inquiry on Portfolio Trade Settlement Batch #412',
        'Can support verify the settlement status of my latest NVDA recurring allocation order?',
        'in_progress', 'medium', 'trading', 'customer', 'cust-longterm-arthur', 'emp-human-elise', 'cust-longterm-arthur', false
    )
ON CONFLICT (ticket_id) DO NOTHING;

--------------------------------------------------------------------------------
-- Initial Ticket Comments
--------------------------------------------------------------------------------
INSERT INTO ops.ticket_comments (comment_id, ticket_id, author_username, comment, is_internal)
VALUES 
    (
        '018e0000-0000-7000-8000-000000000050',
        '018e0000-0000-7000-8000-000000000005',
        'emp-human-elise',
        'INTERNAL ONLY: Reconciled clearing ledger with Mock Broker. Settlement batch matched successfully.',
        true
    ),
    (
        '018e0000-0000-7000-8000-000000000051',
        '018e0000-0000-7000-8000-000000000005',
        'emp-human-elise',
        'Hello Arthur! Your NVDA order settlement has been verified and fully credited to your OCI Alpha Growth Fund portfolio.',
        false
    )
ON CONFLICT (comment_id) DO NOTHING;

--------------------------------------------------------------------------------
-- Seed Initial Review Cycles (Multi-Phase Performance Reviews & Veterinary Audits)
--------------------------------------------------------------------------------
INSERT INTO workforce.review_cycles (
    review_id, employee_id, review_type, review_period, phase, scheduled_for, status,
    reviewer_id, score, notes, self_appraisal, self_score, manager_assessment, manager_score,
    kpi_ratings, wellness_audit_score, habitat_recommendations, clinical_auditor_id, completed_at
) VALUES 
    (
        '018e0000-0000-7000-8000-000000000020',
        '018f3a9a-e001-7000-8000-000000000001',
        'feline_health_assessment',
        '2026-Q3',
        'completed',
        '2026-10-12',
        'completed',
        '018f3a9a-e004-7000-8000-000000000004',
        4.95,
        'Superb whisker symmetry, resting purr acoustics 92dB, optimal alpha sunbeam positioning.',
        'Maintained executive posture on alpha perch; nap duration exceeded target by 12%.',
        5.00,
        'Executive feline leadership has stabilized market confidence across retail portfolios.',
        4.95,
        '{"purr_acoustics": 4.95, "sunbeam_coverage": 5.0, "observation_fidelity": 4.9}'::jsonb,
        4.95,
        'Increase sunlight perch access in Habitat Alpha; maintain lasagna moderation protocol.',
        '018f3a9a-e004-7000-8000-000000000004',
        CURRENT_TIMESTAMP
    ),
    (
        '018e0000-0000-7000-8000-000000000021',
        '018f3a9a-e002-7000-8000-000000000002',
        'feline_health_assessment',
        '2026-Q3',
        'manager_review',
        '2026-10-18',
        'in_progress',
        '018f3a9a-e004-7000-8000-000000000004',
        NULL,
        'Quarterly weight check and alpha perch mobility audit',
        'Climbed perch 42 times this week; verified perimeter integrity.',
        4.80,
        NULL,
        NULL,
        '{"perch_agility": 4.85, "catnip_temperance": 4.75}'::jsonb,
        NULL,
        NULL,
        '018f3a9a-e004-7000-8000-000000000004',
        NULL
    ),
    (
        '018e0000-0000-7000-8000-000000000022',
        '018f3a9a-e003-7000-8000-000000000003',
        'performance',
        '2026-Q3',
        'self_appraisal',
        '2026-10-22',
        'scheduled',
        '018f3a9a-e001-7000-8000-000000000001',
        NULL,
        'Joint human-feline HR operations & catnip compliance review',
        'Automated developer provisioning across Forgejo, Nextcloud, and Snipe-IT.',
        4.85,
        NULL,
        NULL,
        '{"onboarding_efficiency": 4.9, "compliance_adherence": 4.8}'::jsonb,
        NULL,
        NULL,
        NULL,
        NULL
    )
ON CONFLICT (review_id) DO NOTHING;

--------------------------------------------------------------------------------
-- Seed Executive Management Reports
--------------------------------------------------------------------------------
INSERT INTO workforce.executive_reports (
    report_id, period, title, status,
    total_revenue, operating_expenses, payroll_expenses, net_profit,
    operational_burn, cash_reserves, runway_months,
    total_headcount, human_headcount, feline_headcount,
    attendance_rate, overtime_hours, timesheets_approved,
    webdav_path_pdf, webdav_path_csv, generated_by
) VALUES
    (
        'rep-2026-08',
        '2026-08',
        'OCI Executive Management Report - August 2026',
        'archived',
        1180000.00, 310000.00, 395000.00, 475000.00,
        705000.00, 9200000.00, 13.05,
        11, 9, 2,
        98.80, 38.00, 16,
        '/remote.php/dav/files/admin/Executive/Reports/2026-08/executive_summary_report_2026-08.pdf',
        '/remote.php/dav/files/admin/Executive/Reports/2026-08/executive_summary_report_2026-08.csv',
        'system-reporting-worker'
    ),
    (
        'rep-2026-09',
        '2026-09',
        'OCI Executive Management Report - September 2026',
        'archived',
        1240000.00, 335000.00, 410000.00, 495000.00,
        745000.00, 9650000.00, 12.95,
        12, 10, 2,
        99.10, 41.50, 18,
        '/remote.php/dav/files/admin/Executive/Reports/2026-09/executive_summary_report_2026-09.pdf',
        '/remote.php/dav/files/admin/Executive/Reports/2026-09/executive_summary_report_2026-09.csv',
        'system-reporting-worker'
    )
ON CONFLICT (report_id) DO NOTHING;
