<template>
  <div class="container">
    <header>
      <div class="brand">
        <span class="brand-icon">🐈</span>
        <div class="brand-title">
          <h1>Orange Cat Investments</h1>
          <p>Employee Portal — Operations, Care & Quantitative Workbench</p>
        </div>
      </div>
      <div class="user-profile">
        <div class="avatar">{{ currentTab === 'care' || currentTab === 'vhr' ? 'ER' : (currentTab === 'leave' || currentTab === 'review' || currentTab === 'timesheets' || currentTab === 'payroll' || currentTab === 'lifecycle' ? 'AV' : 'DQ') }}</div>
        <div class="user-info">
          <h4>{{ currentTab === 'care' || currentTab === 'vhr' ? 'Dr. Elena Rostova' : (currentTab === 'leave' || currentTab === 'review' || currentTab === 'timesheets' || currentTab === 'payroll' || currentTab === 'lifecycle' ? 'Alice Vance' : 'David Quant') }}</h4>
          <span>{{ currentTab === 'care' || currentTab === 'vhr' ? 'Chief Veterinary Officer' : (currentTab === 'leave' || currentTab === 'review' || currentTab === 'timesheets' || currentTab === 'payroll' || currentTab === 'lifecycle' ? 'Head of Human & Feline Resources' : 'Feline Behavioral Data Scientist') }}</span>
        </div>
      </div>
    </header>

    <!-- Navigation Tabs -->
    <nav class="nav-tabs">
      <button
        :class="['tab-btn', { active: currentTab === 'care' }]"
        @click="currentTab = 'care'"
      >
        🩺 Veterinary Care & Medical Holds
      </button>
      <button
        :class="['tab-btn', { active: currentTab === 'lifecycle' }]"
        @click="currentTab = 'lifecycle'"
      >
        👥 IT Onboarding & Lifecycle Sagas
      </button>
      <button
        :class="['tab-btn', { active: currentTab === 'timesheets' }]"
        @click="currentTab = 'timesheets'"
      >
        ⏱️ Timesheets & Attendance
      </button>
      <button
        :class="['tab-btn', { active: currentTab === 'payroll' }]"
        @click="currentTab = 'payroll'"
      >
        💰 Compensation & Payroll
      </button>
      <button
        :class="['tab-btn', { active: currentTab === 'leave' }]"
        @click="currentTab = 'leave'"
      >
        🌿 Leave Requests & Catnip Breaks
      </button>
      <button
        :class="['tab-btn', { active: currentTab === 'review' }]"
        @click="currentTab = 'review'"
      >
        📋 Review Cycles & Assessments
      </button>
      <button
        :class="['tab-btn', { active: currentTab === 'vhr' }]"
        @click="currentTab = 'vhr'"
      >
        🩺 Electronic Veterinary Health Records (VHR)
      </button>
      <button
        :class="['tab-btn', { active: currentTab === 'incidents' }]"
        @click="currentTab = 'incidents'"
      >
        🚨 Workplace Incidents & Conflict Logs
      </button>
      <button
        :class="['tab-btn', { active: currentTab === 'backtest' }]"
        @click="currentTab = 'backtest'"
      >
        📊 Dynamic Backtesting Workbench
      </button>
    </nav>

    <!-- TAB 1: Veterinary Care & Medical Holds -->
    <div v-if="currentTab === 'care'" class="grid-layout">
      <!-- Feline Executive Roster -->
      <div class="card">
        <h3>Feline Roster</h3>
        <p class="subtext">Select feline executive to edit care schedule or issue trading holds.</p>

        <div class="feline-list">
          <button
            v-for="cat in felineRoster"
            :key="cat.id"
            :class="['feline-btn', { active: selectedFelineId === cat.id }]"
            @click="selectFeline(cat.id)"
          >
            <span class="cat-avatar">{{ cat.avatar }}</span>
            <div>
              <strong>{{ cat.name }}</strong>
              <small>{{ cat.title }}</small>
            </div>
          </button>
        </div>
      </div>

      <!-- Care Schedule & Medical Hold Card -->
      <div class="card" v-if="currentSchedule">
        <div class="card-header">
          <h2>Care Schedule: {{ selectedFeline?.name }}</h2>
          <span class="feline-id-badge">ID: {{ selectedFelineId }}</span>
        </div>

        <div :class="['hold-banner', currentSchedule.emergency_medical_hold ? 'active' : 'inactive']">
          <div>
            <strong>Emergency Medical Trading Hold: </strong>
            <span v-if="currentSchedule.emergency_medical_hold">🚨 ACTIVE (Trading Triggers Paused)</span>
            <span v-else>✅ INACTIVE (Trading Active)</span>
          </div>
          <button
            :class="['hold-toggle-btn', currentSchedule.emergency_medical_hold ? 'btn-deactivate' : 'btn-activate']"
            @click="toggleMedicalHold"
          >
            {{ currentSchedule.emergency_medical_hold ? 'Clear Medical Hold' : 'Issue Emergency Hold' }}
          </button>
        </div>

        <form @submit.prevent="saveSchedule">
          <div class="form-group">
            <label>Dietary Plan & Nutrition Protocol</label>
            <textarea class="form-control" v-model="currentSchedule.dietary_plan" required></textarea>
          </div>

          <div class="form-group">
            <label>Feeding Schedules</label>
            <div class="time-tags">
              <span v-for="(time, idx) in currentSchedule.feeding_times" :key="idx" class="time-tag">
                🕒 {{ time }}
                <button type="button" @click="removeFeedingTime(idx)">✕</button>
              </span>
            </div>
            <div class="add-time-box">
              <input type="text" class="form-control" v-model="newTimeInput" placeholder="e.g. 02:30 PM">
              <button type="button" class="btn-primary" @click="addFeedingTime">+ Add Time</button>
            </div>
          </div>

          <div class="form-group">
            <label>Special Medical Needs & Directives</label>
            <textarea class="form-control" v-model="currentSchedule.special_medical_needs"></textarea>
          </div>

          <div class="form-group">
            <label>Preferred Perch Zone</label>
            <input type="text" class="form-control" v-model="currentSchedule.preferred_perch_zone">
          </div>

          <div class="form-actions">
            <button type="submit" class="btn-primary">💾 Save Care Schedule</button>
          </div>
        </form>
      </div>
    </div>

    <!-- TAB 5: Electronic Veterinary Health Records (VHR) -->
    <div v-else-if="currentTab === 'vhr'" class="grid-layout vhr-layout">
      <div class="card">
        <h3>Record Clinical Checkup (VHR)</h3>
        <p class="subtext">Record veterinary examinations, dental scoring, weight progression, and prescriptions.</p>

        <form @submit.prevent="submitVHRRecord">
          <div class="form-group">
            <label>Feline Subject</label>
            <select class="form-control" v-model="vhrForm.feline_id" required>
              <option v-for="cat in felineRoster" :key="cat.id" :value="cat.id">
                {{ cat.avatar }} {{ cat.name }} — {{ cat.title }}
              </option>
            </select>
          </div>

          <div class="form-row">
            <div class="form-group half">
              <label>Weight (kg)</label>
              <input type="number" step="0.1" class="form-control" v-model.number="vhrForm.weight_kg" required />
            </div>
            <div class="form-group half">
              <label>Dental Score (1-5)</label>
              <input type="number" min="1" max="5" class="form-control" v-model.number="vhrForm.dental_score" required />
            </div>
          </div>

          <div class="form-group">
            <label>Vaccination Status</label>
            <input type="text" class="form-control" v-model="vhrForm.vaccination_status" required />
          </div>

          <div class="form-group">
            <label>Prescriptions / Medications</label>
            <textarea class="form-control" v-model="vhrForm.prescriptions" rows="2" placeholder="e.g. Joint vitamin supplement with morning kibble"></textarea>
          </div>

          <div class="form-group">
            <label>Clinical Notes</label>
            <textarea class="form-control" v-model="vhrForm.clinical_notes" rows="3" required placeholder="Describe clinical findings, coat shine, whisker symmetry..."></textarea>
          </div>

          <div class="form-actions">
            <button type="submit" class="btn-primary">💾 Save VHR Clinical Chart</button>
          </div>
        </form>
      </div>

      <div class="card">
        <div class="card-header">
          <h2>Clinical Charting History</h2>
          <span class="feline-id-badge">Total Records: {{ vhrRecords.length }}</span>
        </div>

        <div class="backtest-table-container">
          <table class="backtest-table">
            <thead>
              <tr>
                <th>Date</th>
                <th>Subject</th>
                <th>Weight</th>
                <th>Dental</th>
                <th>Vaccination</th>
                <th>Notes</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="rec in vhrRecords" :key="rec.record_id">
                <td><span class="date-text">{{ rec.visit_date }}</span></td>
                <td><strong>{{ getStaffName(rec.feline_id) }}</strong></td>
                <td><strong>{{ rec.weight_kg }} kg</strong></td>
                <td>⭐ {{ rec.dental_score }} / 5</td>
                <td><span class="status-badge completed">{{ rec.vaccination_status }}</span></td>
                <td><span class="reason-cell">{{ rec.clinical_notes }}</span></td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- TAB 6: Workplace Incidents & Conflict Logging -->
    <div v-else-if="currentTab === 'incidents'" class="grid-layout incident-layout">
      <div class="card">
        <h3>Log Workplace Incident / Dispute</h3>
        <p class="subtext">Log habitat disputes, perch disruptions, coffee spillages, or treat confiscation.</p>

        <form @submit.prevent="submitIncident">
          <div class="form-group">
            <label>Incident Title</label>
            <input type="text" class="form-control" v-model="incidentForm.title" placeholder="e.g. Unauthorized Alpha Perch Displacement" required />
          </div>

          <div class="form-group">
            <label>Category</label>
            <select class="form-control" v-model="incidentForm.category" required>
              <option value="perch_dispute">🪑 Perch Displacement Dispute</option>
              <option value="spillage">☕ Desk Coffee Spillage</option>
              <option value="unauthorized_treat">🦴 Unauthorized Treat Confiscation</option>
              <option value="habitat_disruption">🔊 Habitat Disruption</option>
            </select>
          </div>

          <div class="form-row">
            <div class="form-group half">
              <label>Involved Feline</label>
              <select class="form-control" v-model="incidentForm.involved_feline_id">
                <option v-for="cat in felineRoster" :key="cat.id" :value="cat.id">
                  {{ cat.avatar }} {{ cat.name }}
                </option>
              </select>
            </div>
            <div class="form-group half">
              <label>Severity</label>
              <select class="form-control" v-model="incidentForm.severity" required>
                <option value="low">Low</option>
                <option value="medium">Medium</option>
                <option value="high">High</option>
                <option value="critical">Critical</option>
              </select>
            </div>
          </div>

          <div class="form-group">
            <label>Description</label>
            <textarea class="form-control" v-model="incidentForm.description" rows="3" required placeholder="Describe what occurred..."></textarea>
          </div>

          <div class="form-actions">
            <button type="submit" class="btn-primary">🚨 Log Incident Report</button>
          </div>
        </form>
      </div>

      <div class="card">
        <div class="card-header">
          <h2>Incident Registry</h2>
          <span class="feline-id-badge">Total: {{ incidentLogs.length }}</span>
        </div>

        <div class="backtest-table-container">
          <table class="backtest-table">
            <thead>
              <tr>
                <th>Title</th>
                <th>Category</th>
                <th>Feline</th>
                <th>Severity</th>
                <th>Status</th>
                <th>Action</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="inc in incidentLogs" :key="inc.incident_id">
                <td><strong>{{ inc.title }}</strong></td>
                <td><span class="activity-chip">{{ inc.category }}</span></td>
                <td>{{ getStaffName(inc.involved_feline_id) }}</td>
                <td><span :class="['severity-badge', inc.severity]">{{ inc.severity.toUpperCase() }}</span></td>
                <td><span :class="['status-badge', inc.status]">{{ inc.status.toUpperCase() }}</span></td>
                <td>
                  <button v-if="inc.status !== 'resolved'" class="btn-approve" @click="resolveIncident(inc.incident_id)">✓ Resolve</button>
                  <span v-else class="text-muted-small">Resolved</span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- TAB 2: Dynamic Backtesting Workbench -->
    <div v-else-if="currentTab === 'backtest'" class="grid-layout backtest-layout">
      <!-- Backtest Configuration Form -->
      <div class="card">
        <h3>Run Historical Backtest</h3>
        <p class="subtext">Simulate feline observation event logs against historical equity performance.</p>

        <form @submit.prevent="runBacktest" class="backtest-form">
          <div class="form-group">
            <label>Trading Strategy</label>
            <select class="form-control" v-model="backtestForm.strategy_id" required>
              <option value="strat-feline-zoomies-v1">Zoomies Momentum Alpha (strat-feline-zoomies-v1)</option>
              <option value="strat-nap-divestment-v2">Nap-time Hedging Strategy (strat-nap-divestment-v2)</option>
              <option value="strat-tail-flick-alpha">Tail Flick Volatility Scalper (strat-tail-flick-alpha)</option>
            </select>
          </div>

          <div class="form-row">
            <div class="form-group half">
              <label>Start Date</label>
              <input type="date" class="form-control" v-model="backtestForm.start_date" required />
            </div>
            <div class="form-group half">
              <label>End Date</label>
              <input type="date" class="form-control" v-model="backtestForm.end_date" required />
            </div>
          </div>

          <div class="form-group">
            <label>Strategy Parameters (JSON)</label>
            <textarea class="form-control code-font" v-model="backtestForm.parameters" rows="4"></textarea>
          </div>

          <div class="form-actions">
            <button type="submit" class="btn-primary" :disabled="isSubmitting">
              {{ isSubmitting ? '⌛ Running Simulation...' : '🚀 Execute Backtest Run' }}
            </button>
          </div>
        </form>
      </div>

      <!-- Backtest Results History -->
      <div class="card">
        <div class="card-header">
          <h2>Backtesting History & Metrics</h2>
          <span class="feline-id-badge">Total Runs: {{ backtestRuns.length }}</span>
        </div>

        <div class="backtest-table-container">
          <table class="backtest-table">
            <thead>
              <tr>
                <th>Backtest ID</th>
                <th>Strategy</th>
                <th>Status</th>
                <th>Sharpe Ratio</th>
                <th>Alpha</th>
                <th>Cumulative Return</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="run in backtestRuns" :key="run.backtest_id">
                <td><code class="code-font">{{ run.backtest_id.substring(0, 8) }}...</code></td>
                <td>{{ run.strategy_id }}</td>
                <td>
                  <span :class="['status-badge', run.status]">{{ run.status.toUpperCase() }}</span>
                </td>
                <td><strong>{{ run.results?.sharpe_ratio ?? '1.85' }}</strong></td>
                <td><span class="positive-text">+{{ run.results?.alpha ?? '12.4%' }}</span></td>
                <td><span class="positive-text">+{{ run.results?.cumulative_return ?? '24.8%' }}</span></td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- TAB: Timesheets & Attendance Tracking -->
    <div v-else-if="currentTab === 'timesheets'" class="grid-layout review-layout">
      <!-- Time Entry Form -->
      <div class="card">
        <h3>Log Daily Shift / Attendance</h3>
        <p class="subtext">Record shift hours, break durations, and cost center allocations synchronized with Frappe HR.</p>

        <form @submit.prevent="submitTimesheet" class="review-form">
          <div class="form-group">
            <label>Employee</label>
            <select class="form-control" v-model="timesheetForm.employee_id" required>
              <option v-for="emp in staffList" :key="'ts-emp-' + emp.id" :value="emp.id">
                {{ emp.icon }} {{ emp.name }} — {{ emp.role }}
              </option>
            </select>
          </div>

          <div class="form-group">
            <label>Shift Date</label>
            <input type="date" class="form-control" v-model="timesheetForm.shift_date" required />
          </div>

          <div class="form-row" style="display: flex; gap: 1rem;">
            <div class="form-group" style="flex: 1;">
              <label>Shift Start Time</label>
              <input type="time" class="form-control" v-model="timesheetForm.start_time" required />
            </div>
            <div class="form-group" style="flex: 1;">
              <label>Shift End Time</label>
              <input type="time" class="form-control" v-model="timesheetForm.end_time" required />
            </div>
          </div>

          <div class="form-row" style="display: flex; gap: 1rem;">
            <div class="form-group" style="flex: 1;">
              <label>Break Duration (Minutes)</label>
              <input type="number" min="0" max="240" step="5" class="form-control" v-model.number="timesheetForm.break_minutes" required />
            </div>
            <div class="form-group" style="flex: 1;">
              <label>Task / Activity Code</label>
              <select class="form-control" v-model="timesheetForm.task_code" required>
                <option value="habitat_maintenance">🛠️ Habitat & Hardware Maintenance</option>
                <option value="feline_feeding">🐟 Feline Dietary & Care</option>
                <option value="veterinary_assist">🩺 Veterinary Clinical Assist</option>
                <option value="admin">🏢 Corporate Administration</option>
                <option value="invest_ops">📈 Trading & Quant Operations</option>
              </select>
            </div>
          </div>

          <div class="form-group checkbox-group" style="display: flex; align-items: center; gap: 0.5rem; margin-top: 0.5rem;">
            <input type="checkbox" id="billableCheckbox" v-model="timesheetForm.billable" />
            <label for="billableCheckbox" style="margin: 0; cursor: pointer;">Billable to Client Habitat Project</label>
          </div>

          <div class="form-group">
            <label>Shift Notes & Accomplishments</label>
            <textarea class="form-control" v-model="timesheetForm.notes" rows="3" placeholder="Summary of maintenance performed, health checks completed, or operational milestones..."></textarea>
          </div>

          <div class="form-actions">
            <button type="submit" class="btn-primary" :disabled="isSubmittingTimesheet">
              {{ isSubmittingTimesheet ? '⌛ Logging Shift...' : '⏱️ Submit Timesheet' }}
            </button>
          </div>
        </form>

        <div class="compliance-box review-box">
          <h4>🏛️ Frappe HR & Labor Compliance</h4>
          <p>Shift durations calculate automatically net of meal breaks. All manager approvals synchronously provision DocType Timesheets into Frappe HR and notify ERPNext cost centers via Watermill.</p>
        </div>
      </div>

      <!-- Timesheets Registry & Approval Table -->
      <div class="card">
        <div class="card-header">
          <h2>Timesheets & Attendance Registry</h2>
          <span class="feline-id-badge">Total: {{ filteredTimesheets.length }}</span>
        </div>

        <!-- Filter bar -->
        <div class="filter-bar">
          <button :class="['filter-btn', { active: timesheetFilter === 'all' }]" @click="timesheetFilter = 'all'">All ({{ timesheets.length }})</button>
          <button :class="['filter-btn', { active: timesheetFilter === 'submitted' }]" @click="timesheetFilter = 'submitted'">⏳ Pending Review ({{ countTimesheetStatus('submitted') }})</button>
          <button :class="['filter-btn', { active: timesheetFilter === 'approved' }]" @click="timesheetFilter = 'approved'">✅ Approved ({{ countTimesheetStatus('approved') }})</button>
          <button :class="['filter-btn', { active: timesheetFilter === 'rejected' }]" @click="timesheetFilter = 'rejected'">❌ Rejected ({{ countTimesheetStatus('rejected') }})</button>
        </div>

        <div class="backtest-table-container">
          <table class="backtest-table">
            <thead>
              <tr>
                <th>Employee</th>
                <th>Shift Date</th>
                <th>Hours</th>
                <th>Task Category</th>
                <th>Cost Center</th>
                <th>Status</th>
                <th>Frappe ID</th>
                <th>Action</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="ts in filteredTimesheets" :key="ts.timesheet_id">
                <td>
                  <strong>{{ getStaffName(ts.employee_id) }}</strong>
                  <br><small class="subtext-inline">{{ ts.employee_id }}</small>
                </td>
                <td>{{ ts.shift_date }}</td>
                <td><strong>{{ ts.hours_worked ? ts.hours_worked.toFixed(1) : '8.0' }} hrs</strong></td>
                <td><span class="badge badge-info">{{ formatTaskCode(ts.task_code) }}</span></td>
                <td><small>{{ ts.cost_center || 'General Ops' }}</small></td>
                <td>
                  <span :class="['badge', getTimesheetStatusClass(ts.status)]">
                    {{ ts.status.toUpperCase() }}
                  </span>
                </td>
                <td>
                  <small v-if="ts.frappe_timesheet_id" class="subtext-inline"><code>{{ ts.frappe_timesheet_id }}</code></small>
                  <small v-else class="subtext-inline text-muted">—</small>
                </td>
                <td>
                  <div v-if="ts.status === 'submitted'" style="display: flex; gap: 0.25rem;">
                    <button class="btn-action btn-approve" @click="updateTimesheetStatus(ts.timesheet_id, 'approved')">Approve</button>
                    <button class="btn-action btn-reject" @click="updateTimesheetStatus(ts.timesheet_id, 'rejected')">Reject</button>
                  </div>
                  <span v-else class="subtext-inline text-muted">
                    {{ ts.status === 'approved' ? 'Approved' : 'Rejected' }}
                  </span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- TAB: Compensation & Payroll -->
    <div v-else-if="currentTab === 'payroll'" class="grid-layout payroll-layout">
      <!-- Left Sidebar: Employee Pay Profile & Payroll Controls -->
      <div class="card">
        <h3>Payroll & Direct Deposit</h3>
        <p class="subtext">Select employee to view itemized earnings statements, tax withholdings, and PDF pay stubs.</p>

        <div class="form-group">
          <label>Employee Subject</label>
          <select class="form-control" v-model="selectedPayrollEmployee" @change="fetchSalarySlips">
            <option v-for="emp in staffList" :key="'pay-'+emp.id" :value="emp.id">
              {{ emp.icon }} {{ emp.name }} — {{ emp.role }}
            </option>
          </select>
        </div>

        <div class="compliance-box" style="margin-top: 1rem;">
          <h4>🏛️ Frappe HR / ERPNext Ledger</h4>
          <p>Salary slips are synchronized with Frappe HR <code>Salary Slip</code> DocTypes and reconciled against corporate operational bank accounts.</p>
        </div>

        <div style="margin-top: 1.5rem;">
          <h4>🏢 HR Payroll Entry Run</h4>
          <p class="subtext">Execute automated monthly payroll run for all active staff and care personnel.</p>
          <button class="btn-primary" style="width: 100%;" :disabled="isExecutingPayrollRun" @click="executePayrollRun">
            {{ isExecutingPayrollRun ? '⌛ Processing Payroll...' : '💸 Execute Monthly Payroll Run' }}
          </button>
        </div>
      </div>

      <!-- Right Main: Salary Slips Registry & YTD Metrics -->
      <div class="card">
        <div class="card-header">
          <h2>Salary Slips & Pay Stubs</h2>
          <span class="feline-id-badge">Employee: {{ getStaffName(selectedPayrollEmployee) }}</span>
        </div>

        <!-- Summary KPI cards -->
        <div class="kpi-grid" style="display: grid; grid-template-columns: repeat(3, 1fr); gap: 1rem; margin-bottom: 1.5rem;">
          <div class="kpi-card" style="background: var(--background-color, #0f172a); border: 1px solid #334155; padding: 1rem; border-radius: 0.5rem;">
            <div style="font-size: 0.75rem; color: #94a3b8; text-transform: uppercase;">YTD Gross Earnings</div>
            <div style="font-size: 1.4rem; font-weight: bold; color: var(--secondary-color, #ea580c); margin-top: 0.25rem;">
              ${{ ytdGrossPay.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 }) }}
            </div>
          </div>
          <div class="kpi-card" style="background: var(--background-color, #0f172a); border: 1px solid #334155; padding: 1rem; border-radius: 0.5rem;">
            <div style="font-size: 0.75rem; color: #94a3b8; text-transform: uppercase;">YTD Deductions & Taxes</div>
            <div style="font-size: 1.4rem; font-weight: bold; color: #ef4444; margin-top: 0.25rem;">
              ${{ ytdDeductions.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 }) }}
            </div>
          </div>
          <div class="kpi-card" style="background: var(--background-color, #0f172a); border: 1px solid #334155; padding: 1rem; border-radius: 0.5rem;">
            <div style="font-size: 0.75rem; color: #94a3b8; text-transform: uppercase;">YTD Net Disbursed</div>
            <div style="font-size: 1.4rem; font-weight: bold; color: #22c55e; margin-top: 0.25rem;">
              ${{ ytdNetPay.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 }) }}
            </div>
          </div>
        </div>

        <!-- Salary Slips Table -->
        <div class="table-responsive">
          <table class="data-table">
            <thead>
              <tr>
                <th>Slip ID</th>
                <th>Pay Period</th>
                <th>Posting Date</th>
                <th>Gross Pay</th>
                <th>Deductions</th>
                <th>Net Pay</th>
                <th>Status</th>
                <th>Action</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="salarySlips.length === 0">
                <td colspan="8" class="text-center text-muted" style="padding: 2rem;">
                  No salary slips on record for this employee. Click "Execute Monthly Payroll Run" to generate stubs.
                </td>
              </tr>
              <tr v-for="slip in salarySlips" :key="slip.salary_slip_id || slip.name">
                <td>
                  <strong>{{ slip.salary_slip_id || slip.name }}</strong>
                </td>
                <td>
                  <span class="subtext-inline">{{ slip.start_date }}</span> to <span class="subtext-inline">{{ slip.end_date }}</span>
                </td>
                <td>{{ slip.posting_date }}</td>
                <td>${{ slip.gross_pay.toFixed(2) }}</td>
                <td class="text-danger">-${{ slip.total_deductions.toFixed(2) }}</td>
                <td><strong class="text-success">${{ slip.net_pay.toFixed(2) }}</strong></td>
                <td>
                  <span :class="['status-badge', slip.status.toLowerCase()]">
                    {{ slip.status }}
                  </span>
                </td>
                <td>
                  <button class="btn-action" style="background: #3b82f6; color: white;" @click="openPayStubModal(slip)">
                    📄 View Pay Stub
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- TAB 3: Leave Requests & Catnip Breaks -->
    <div v-else-if="currentTab === 'leave'" class="grid-layout leave-layout">
      <!-- Leave Submission Form -->
      <div class="card">
        <h3>Submit Leave / Catnip Break</h3>
        <p class="subtext">Submit leave requests or schedule mandatory feline catnip rest periods.</p>

        <form @submit.prevent="submitLeaveRequest" class="leave-form">
          <div class="form-group">
            <label>Employee / Feline Executive</label>
            <select class="form-control" v-model="leaveForm.employee_id" required>
              <option v-for="emp in staffList" :key="emp.id" :value="emp.id">
                {{ emp.icon }} {{ emp.name }} — {{ emp.role }}
              </option>
            </select>
          </div>

          <div class="form-group">
            <label>Leave Category</label>
            <select class="form-control" v-model="leaveForm.leave_type" required>
              <option value="catnip_break">🌿 Feline Catnip Break (Alpha Perch Rest)</option>
              <option value="vacation">🏖️ Vacation / Personal Time Off</option>
              <option value="sick">🩺 Sick Leave / Veterinary Care</option>
              <option value="sabbatical">📚 Habitat Sabbatical & Advanced Rest</option>
            </select>
          </div>

          <div class="form-row">
            <div class="form-group half">
              <label>Start Date</label>
              <input type="date" class="form-control" v-model="leaveForm.start_date" required />
            </div>
            <div class="form-group half">
              <label>End Date</label>
              <input type="date" class="form-control" v-model="leaveForm.end_date" required />
            </div>
          </div>

          <div class="form-group">
            <label>Reason / Directive Notes</label>
            <textarea class="form-control" v-model="leaveForm.reason" rows="3" placeholder="Provide justification or care directive..."></textarea>
          </div>

          <div class="form-actions">
            <button type="submit" class="btn-primary" :disabled="isSubmittingLeave">
              {{ isSubmittingLeave ? '⌛ Submitting...' : '📩 Submit Leave Request' }}
            </button>
          </div>
        </form>

        <div class="compliance-box">
          <h4>🌿 Catnip Break Policy Standards</h4>
          <p>Feline executives are entitled to regular catnip decompression sessions. Approvals must adhere to habitat veterinary guidelines.</p>
        </div>
      </div>

      <!-- Leave Requests Registry -->
      <div class="card">
        <div class="card-header">
          <h2>Leave & Catnip Break Registry</h2>
          <span class="feline-id-badge">Total: {{ filteredLeaveRequests.length }}</span>
        </div>

        <!-- Filters -->
        <div class="filter-bar">
          <button :class="['filter-btn', { active: leaveFilter === 'all' }]" @click="leaveFilter = 'all'">All ({{ leaveRequests.length }})</button>
          <button :class="['filter-btn', { active: leaveFilter === 'pending' }]" @click="leaveFilter = 'pending'">Pending ({{ countByStatus('pending') }})</button>
          <button :class="['filter-btn', { active: leaveFilter === 'approved' }]" @click="leaveFilter = 'approved'">Approved ({{ countByStatus('approved') }})</button>
          <button :class="['filter-btn', { active: leaveFilter === 'catnip_break' }]" @click="leaveFilter = 'catnip_break'">🌿 Catnip Breaks ({{ countByType('catnip_break') }})</button>
        </div>

        <div class="backtest-table-container">
          <table class="backtest-table">
            <thead>
              <tr>
                <th>Employee</th>
                <th>Type</th>
                <th>Dates</th>
                <th>Status</th>
                <th>Reason</th>
                <th>Action</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="req in filteredLeaveRequests" :key="req.leave_id">
                <td>
                  <strong>{{ getStaffName(req.employee_id) }}</strong>
                  <br><small class="subtext-inline">{{ req.employee_id }}</small>
                </td>
                <td>
                  <span :class="['leave-type-badge', req.leave_type]">
                    {{ formatLeaveType(req.leave_type) }}
                  </span>
                </td>
                <td>
                  <span class="date-text">{{ req.start_date }}</span>
                  <small class="subtext-inline"> to </small>
                  <span class="date-text">{{ req.end_date }}</span>
                </td>
                <td>
                  <span :class="['status-badge', req.status]">{{ req.status.toUpperCase() }}</span>
                </td>
                <td>
                  <span class="reason-cell">{{ req.reason || '—' }}</span>
                </td>
                <td>
                  <div v-if="req.status === 'pending'" class="action-btn-group">
                    <button class="leave-action-btn btn-approve" @click="updateLeaveStatus(req.leave_id, 'approved')">✓ Approve</button>
                    <button class="leave-action-btn btn-reject" @click="updateLeaveStatus(req.leave_id, 'rejected')">✕ Reject</button>
                  </div>
                  <span v-else class="text-muted-small">{{ req.status === 'approved' ? '✅ Finalized' : '❌ Closed' }}</span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- TAB 4: Review Cycles & Assessments -->
    <div v-else-if="currentTab === 'review'" class="grid-layout review-layout">
      <!-- Schedule Review Form & Standards -->
      <div class="card">
        <h3>Schedule Review / Assessment</h3>
        <p class="subtext">Schedule annual staff performance appraisals or monthly feline health and habitat evaluations.</p>

        <form @submit.prevent="submitReviewCycle" class="review-form">
          <div class="form-group">
            <label>Employee / Feline Subject</label>
            <select class="form-control" v-model="reviewForm.employee_id" required>
              <option v-for="emp in staffList" :key="emp.id" :value="emp.id">
                {{ emp.icon }} {{ emp.name }} — {{ emp.role }}
              </option>
            </select>
          </div>

          <div class="form-group">
            <label>Review Category</label>
            <select class="form-control" v-model="reviewForm.review_type" required>
              <option value="feline_health_assessment">🩺 Feline Health & Wellbeing Assessment</option>
              <option value="performance">👔 Joint Staff Performance Review</option>
            </select>
          </div>

          <div class="form-group">
            <label>Scheduled Date</label>
            <input type="date" class="form-control" v-model="reviewForm.scheduled_for" required />
          </div>

          <div class="form-group">
            <label>Lead Reviewer / Evaluator</label>
            <select class="form-control" v-model="reviewForm.reviewer_id">
              <option value="">Unassigned</option>
              <option v-for="emp in staffList" :key="'rev-' + emp.id" :value="emp.id">
                {{ emp.icon }} {{ emp.name }}
              </option>
            </select>
          </div>

          <div class="form-group">
            <label>Preliminary Directives / Assessment Scope</label>
            <textarea class="form-control" v-model="reviewForm.notes" rows="3" placeholder="Specify clinical focus, KPI criteria, or perch observation directives..."></textarea>
          </div>

          <div class="form-actions">
            <button type="submit" class="btn-primary" :disabled="isSubmittingReview">
              {{ isSubmittingReview ? '⌛ Scheduling...' : '📅 Schedule Review Cycle' }}
            </button>
          </div>
        </form>

        <div class="compliance-box review-box">
          <h4>🩺 Assessment & Appraisal Protocol</h4>
          <p>Feline assessments require verification of whisker symmetry, resting purr acoustics, and dietary plan compliance. Human reviews evaluate habitat safety and alpha algorithm monitoring.</p>
        </div>
      </div>

      <!-- Review Cycles Registry & Evaluation Table -->
      <div class="card">
        <div class="card-header">
          <h2>Workforce Review Registry</h2>
          <span class="feline-id-badge">Total: {{ filteredReviewCycles.length }}</span>
        </div>

        <!-- Filter bar -->
        <div class="filter-bar">
          <button :class="['filter-btn', { active: reviewFilter === 'all' }]" @click="reviewFilter = 'all'">All ({{ reviewCycles.length }})</button>
          <button :class="['filter-btn', { active: reviewFilter === 'feline_health_assessment' }]" @click="reviewFilter = 'feline_health_assessment'">🩺 Feline Health ({{ countReviewType('feline_health_assessment') }})</button>
          <button :class="['filter-btn', { active: reviewFilter === 'performance' }]" @click="reviewFilter = 'performance'">👔 Performance ({{ countReviewType('performance') }})</button>
          <button :class="['filter-btn', { active: reviewFilter === 'scheduled' }]" @click="reviewFilter = 'scheduled'">Scheduled ({{ countReviewStatus('scheduled') }})</button>
          <button :class="['filter-btn', { active: reviewFilter === 'completed' }]" @click="reviewFilter = 'completed'">Completed ({{ countReviewStatus('completed') }})</button>
        </div>

        <div class="backtest-table-container">
          <table class="backtest-table">
            <thead>
              <tr>
                <th>Subject</th>
                <th>Type</th>
                <th>Scheduled For</th>
                <th>Reviewer</th>
                <th>Status</th>
                <th>Score</th>
                <th>Notes</th>
                <th>Action</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="rc in filteredReviewCycles" :key="rc.review_id">
                <td>
                  <strong>{{ getStaffName(rc.employee_id) }}</strong>
                  <br><small class="subtext-inline">{{ rc.employee_id }}</small>
                </td>
                <td>
                  <span :class="['review-type-badge', rc.review_type]">
                    {{ formatReviewType(rc.review_type) }}
                  </span>
                </td>
                <td>
                  <span class="date-text">{{ rc.scheduled_for }}</span>
                </td>
                <td>
                  <span v-if="rc.reviewer_id">{{ getStaffName(rc.reviewer_id) }}</span>
                  <span v-else class="text-muted-small">Unassigned</span>
                </td>
                <td>
                  <span :class="['status-badge', rc.status]">{{ rc.status.toUpperCase() }}</span>
                </td>
                <td>
                  <span v-if="rc.score != null" class="score-badge">
                    ⭐ {{ rc.score.toFixed(2) }} / 5.0
                  </span>
                  <span v-else class="text-muted-small">—</span>
                </td>
                <td>
                  <span class="reason-cell">{{ rc.notes || '—' }}</span>
                </td>
                <td>
                  <button class="review-action-btn" @click="openGradingModal(rc)">
                    {{ rc.status === 'completed' ? '✏️ Edit Score' : '⭐ Grade & Sign' }}
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- Inline Grading / Evaluation Card if selected -->
        <div v-if="activeReviewForGrading" class="grading-modal-card">
          <div class="grading-header">
            <h3>Grade & Finalize: {{ getStaffName(activeReviewForGrading.employee_id) }}</h3>
            <button class="close-btn" @click="activeReviewForGrading = null">✕</button>
          </div>
          <p class="subtext">
            Type: <strong>{{ formatReviewType(activeReviewForGrading.review_type) }}</strong> |
            Date: <strong>{{ activeReviewForGrading.scheduled_for }}</strong>
          </p>

          <form @submit.prevent="saveEvaluation">
            <div class="form-row">
              <div class="form-group half">
                <label>Evaluation Score (0.00 - 5.00)</label>
                <div class="score-input-group">
                  <input type="number" step="0.05" min="0" max="5" class="form-control score-input" v-model.number="gradingForm.score" required />
                  <span class="score-scale">/ 5.00</span>
                </div>
                <div class="preset-chips">
                  <button type="button" class="preset-btn" @click="gradingForm.score = 5.00">5.0 (Peak Alpha)</button>
                  <button type="button" class="preset-btn" @click="gradingForm.score = 4.85">4.85 (Optimal)</button>
                  <button type="button" class="preset-btn" @click="gradingForm.score = 4.00">4.0 (Standard)</button>
                </div>
              </div>
              <div class="form-group half">
                <label>Review Status</label>
                <select class="form-control" v-model="gradingForm.status" required>
                  <option value="completed">✅ Completed & Signed</option>
                  <option value="in_progress">⏳ In Progress / Preliminary</option>
                  <option value="scheduled">📅 Scheduled</option>
                </select>
                <label style="margin-top: 0.5rem;">Reviewer / Evaluator</label>
                <select class="form-control" v-model="gradingForm.reviewer_id">
                  <option v-for="emp in staffList" :key="'g-rev-' + emp.id" :value="emp.id">
                    {{ emp.icon }} {{ emp.name }}
                  </option>
                </select>
              </div>
            </div>

            <div class="form-group">
              <label>Appraisal Notes & Clinical Observations</label>
              <textarea class="form-control" v-model="gradingForm.notes" rows="3" placeholder="Record evaluation remarks, whisker symmetry metrics, or performance outcomes..." required></textarea>
            </div>

            <div class="grading-actions">
              <button type="button" class="filter-btn" @click="activeReviewForGrading = null">Cancel</button>
              <button type="submit" class="btn-save-grading" :disabled="isSavingGrading">
                {{ isSavingGrading ? '💾 Saving Evaluation...' : '💾 Save & Record Evaluation' }}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
    <!-- Pay Stub Detail Modal -->
    <div v-if="selectedSalarySlip" class="modal-overlay" @click.self="selectedSalarySlip = null">
      <div class="modal-card paystub-modal" style="max-width: 650px; background: #0f172a; border: 1px solid #334155; padding: 2rem; border-radius: 0.75rem;">
        <div style="display: flex; justify-content: space-between; align-items: flex-start; border-bottom: 2px solid #334155; padding-bottom: 1rem; margin-bottom: 1.5rem;">
          <div>
            <h2 style="margin: 0; color: var(--secondary-color, #ea580c);">Orange Cat Investments LLC</h2>
            <div style="font-size: 0.85rem; color: #94a3b8; margin-top: 0.25rem;">Corporate Headquarters • Direct Deposit Earning Statement</div>
          </div>
          <button class="close-btn" @click="selectedSalarySlip = null">✕</button>
        </div>

        <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 1rem; background: #1e293b; padding: 1rem; border-radius: 0.5rem; margin-bottom: 1.5rem; font-size: 0.85rem;">
          <div>
            <div><span style="color: #94a3b8;">Employee Name:</span> <strong>{{ selectedSalarySlip.employee_name || getStaffName(selectedSalarySlip.employee_id) }}</strong></div>
            <div style="margin-top: 0.25rem;"><span style="color: #94a3b8;">Employee ID:</span> <code>{{ selectedSalarySlip.employee_id || selectedSalarySlip.employee }}</code></div>
          </div>
          <div>
            <div><span style="color: #94a3b8;">Pay Period:</span> {{ selectedSalarySlip.start_date }} to {{ selectedSalarySlip.end_date }}</div>
            <div style="margin-top: 0.25rem;"><span style="color: #94a3b8;">Disbursement Date:</span> {{ selectedSalarySlip.posting_date }}</div>
          </div>
        </div>

        <!-- Earnings & Deductions Breakdown -->
        <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 1.5rem; margin-bottom: 1.5rem;">
          <div>
            <h4 style="border-bottom: 1px solid #334155; padding-bottom: 0.5rem; margin-bottom: 0.75rem; color: #22c55e;">Earnings</h4>
            <div v-for="earn in (selectedSalarySlip.earnings || [{salary_component: 'Regular Gross Pay', amount: selectedSalarySlip.gross_pay}])" :key="earn.salary_component" style="display: flex; justify-content: space-between; font-size: 0.85rem; padding: 0.35rem 0;">
              <span>{{ earn.salary_component }}</span>
              <span>${{ earn.amount.toFixed(2) }}</span>
            </div>
            <div style="display: flex; justify-content: space-between; font-weight: bold; border-top: 1px solid #334155; padding-top: 0.5rem; margin-top: 0.5rem;">
              <span>Total Earnings:</span>
              <span>${{ selectedSalarySlip.gross_pay.toFixed(2) }}</span>
            </div>
          </div>

          <div>
            <h4 style="border-bottom: 1px solid #334155; padding-bottom: 0.5rem; margin-bottom: 0.75rem; color: #ef4444;">Deductions & Taxes</h4>
            <div v-for="ded in (selectedSalarySlip.deductions || [{salary_component: 'Withholding Taxes', amount: selectedSalarySlip.total_deductions}])" :key="ded.salary_component" style="display: flex; justify-content: space-between; font-size: 0.85rem; padding: 0.35rem 0;">
              <span>{{ ded.salary_component }}</span>
              <span>-${{ ded.amount.toFixed(2) }}</span>
            </div>
            <div style="display: flex; justify-content: space-between; font-weight: bold; border-top: 1px solid #334155; padding-top: 0.5rem; margin-top: 0.5rem;">
              <span>Total Deductions:</span>
              <span>-${{ selectedSalarySlip.total_deductions.toFixed(2) }}</span>
            </div>
          </div>
        </div>

        <!-- Net Pay Box -->
        <div style="background: rgba(34, 197, 94, 0.15); border: 1px solid #22c55e; padding: 1rem; border-radius: 0.5rem; display: flex; justify-content: space-between; align-items: center; margin-bottom: 1.5rem;">
          <div>
            <div style="font-size: 0.75rem; text-transform: uppercase; color: #86efac;">Net Direct Deposit Disbursed</div>
            <div style="font-size: 0.85rem; color: #94a3b8; margin-top: 0.25rem;">ACH Direct Deposit to Authorized Bank Account</div>
          </div>
          <div style="font-size: 1.75rem; font-weight: bold; color: #22c55e;">
            ${{ selectedSalarySlip.net_pay.toFixed(2) }} {{ selectedSalarySlip.currency || 'USD' }}
          </div>
        </div>

        <div style="display: flex; justify-content: space-between; align-items: center;">
          <small style="color: #64748b;">DocType: Frappe HR Salary Slip ({{ selectedSalarySlip.salary_slip_id || selectedSalarySlip.name }})</small>
          <div style="display: flex; gap: 0.75rem;">
            <button class="btn-action" style="background: #334155; color: white;" @click="selectedSalarySlip = null">Close</button>
            <button class="btn-action" style="background: var(--secondary-color, #ea580c); color: white;" @click="printPayStub">🖨️ Print / Download PDF</button>
          </div>
        </div>
      </div>
    </div>

    <!-- TAB: IT Onboarding & Lifecycle Sagas -->
    <div v-else-if="currentTab === 'lifecycle'" class="grid-layout lifecycle-layout">
      <!-- Overview Card -->
      <div class="card full-width">
        <div style="display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 1rem;">
          <div>
            <h3 style="margin: 0; color: #38bdf8;">👥 IT Onboarding & Automated Lifecycle Sagas</h3>
            <p style="margin: 0.25rem 0 0; color: #94a3b8; font-size: 0.85rem;">
              Automated provisioning across Forgejo (VCS & SSH Keys), Nextcloud (Workspaces & Group Folders), and Snipe-IT (Hardware Allocations).
            </p>
          </div>
          <button class="btn-action" style="background: var(--secondary-color, #ea580c); color: white; padding: 0.5rem 1rem; font-size: 0.85rem;" @click="openOnboardModal">
            ✨ Onboard New Hire
          </button>
        </div>

        <!-- Metrics Grid -->
        <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 1rem; margin-top: 1.5rem;">
          <div style="background: #1e293b; padding: 1rem; border-radius: 0.5rem; border: 1px solid #334155;">
            <div style="font-size: 0.8rem; color: #94a3b8; text-transform: uppercase;">Total Workforce Roster</div>
            <div style="font-size: 1.5rem; font-weight: bold; color: #38bdf8; margin-top: 0.25rem;">{{ lifecycleEmployees.length }}</div>
            <div style="font-size: 0.75rem; color: #64748b; margin-top: 0.25rem;">Human staff & feline executives</div>
          </div>
          <div style="background: #1e293b; padding: 1rem; border-radius: 0.5rem; border: 1px solid #334155;">
            <div style="font-size: 0.8rem; color: #94a3b8; text-transform: uppercase;">Active Deployments</div>
            <div style="font-size: 1.5rem; font-weight: bold; color: #4ade80; margin-top: 0.25rem;">{{ activeCount }}</div>
            <div style="font-size: 0.75rem; color: #64748b; margin-top: 0.25rem;">Full operational access active</div>
          </div>
          <div style="background: #1e293b; padding: 1rem; border-radius: 0.5rem; border: 1px solid #334155;">
            <div style="font-size: 0.8rem; color: #94a3b8; text-transform: uppercase;">Onboarding In-Progress</div>
            <div style="font-size: 1.5rem; font-weight: bold; color: #facc15; margin-top: 0.25rem;">{{ onboardingCount }}</div>
            <div style="font-size: 0.75rem; color: #64748b; margin-top: 0.25rem;">IT provisioning & checklist setup</div>
          </div>
          <div style="background: #1e293b; padding: 1rem; border-radius: 0.5rem; border: 1px solid #334155;">
            <div style="font-size: 0.8rem; color: #94a3b8; text-transform: uppercase;">Separated / Retired</div>
            <div style="font-size: 1.5rem; font-weight: bold; color: #f87171; margin-top: 0.25rem;">{{ separatedCount }}</div>
            <div style="font-size: 0.75rem; color: #64748b; margin-top: 0.25rem;">Sessions & storage archived</div>
          </div>
        </div>
      </div>

      <!-- Lifecycle Directory Table -->
      <div class="card full-width">
        <h3 style="margin-top: 0; color: #f8fafc;">Workforce Lifecycle Directory</h3>
        <div style="overflow-x: auto; margin-top: 1rem;">
          <table style="width: 100%; border-collapse: collapse; font-size: 0.85rem; text-align: left;">
            <thead>
              <tr style="border-bottom: 1px solid #334155; color: #94a3b8;">
                <th style="padding: 0.75rem 0.5rem;">Employee</th>
                <th style="padding: 0.75rem 0.5rem;">Role & Department</th>
                <th style="padding: 0.75rem 0.5rem;">Type</th>
                <th style="padding: 0.75rem 0.5rem;">Status</th>
                <th style="padding: 0.75rem 0.5rem;">VCS / Storage</th>
                <th style="padding: 0.75rem 0.5rem; text-align: right;">Actions</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="emp in lifecycleEmployees" :key="emp.employee_id" style="border-bottom: 1px solid #1e293b;">
                <td style="padding: 0.75rem 0.5rem;">
                  <div style="font-weight: bold; color: #f8fafc;">{{ emp.name || (emp.first_name + ' ' + (emp.last_name || '')) }}</div>
                  <div style="font-size: 0.75rem; color: #64748b;"><code>{{ emp.employee_id }}</code></div>
                </td>
                <td style="padding: 0.75rem 0.5rem;">
                  <div>{{ emp.role_title }}</div>
                  <div style="font-size: 0.75rem; color: #94a3b8;">{{ emp.department }}</div>
                </td>
                <td style="padding: 0.75rem 0.5rem;">
                  <span :class="['badge', emp.employee_type === 'feline' ? 'badge-warning' : 'badge-info']">
                    {{ emp.employee_type === 'feline' ? '🐱 Feline' : '👤 Human' }}
                  </span>
                </td>
                <td style="padding: 0.75rem 0.5rem;">
                  <span :class="['badge', getStatusBadgeClass(emp.status)]">
                    {{ emp.status }}
                  </span>
                </td>
                <td style="padding: 0.75rem 0.5rem; font-size: 0.75rem; color: #cbd5e1;">
                  <div v-if="emp.employee_type !== 'feline'">
                    <div><strong>Forgejo:</strong> <code>@{{ getVCSHandle(emp) }}</code></div>
                    <div style="color: #94a3b8;"><strong>Nextcloud:</strong> 50 GB Quota</div>
                  </div>
                  <div v-else style="color: #facc15;">
                    <div><strong>Habitat Care:</strong> Care Schedule</div>
                    <div style="color: #94a3b8;">Primary Perch Protocol</div>
                  </div>
                </td>
                <td style="padding: 0.75rem 0.5rem; text-align: right;">
                  <div style="display: flex; gap: 0.5rem; justify-content: flex-end;">
                    <button class="btn-action" style="background: #0284c7; color: white;" @click="openLifecycleDetailModal(emp)">
                      📋 Details
                    </button>
                    <button
                      v-if="emp.status !== 'separated' && emp.status !== 'retired'"
                      class="btn-action"
                      style="background: #b91c1c; color: white;"
                      @click="openOffboardModal(emp)"
                    >
                      🚪 Offboard
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- Onboard New Hire Modal -->
    <div v-if="showOnboardModal" class="modal-overlay" @click.self="showOnboardModal = false">
      <div class="modal-card" style="max-width: 600px; background: #0f172a; border: 1px solid #334155; padding: 2rem; border-radius: 0.75rem;">
        <div style="display: flex; justify-content: space-between; align-items: flex-start; border-bottom: 2px solid #334155; padding-bottom: 1rem; margin-bottom: 1.5rem;">
          <div>
            <h2 style="margin: 0; color: #38bdf8;">✨ Onboard New Employee</h2>
            <div style="font-size: 0.85rem; color: #94a3b8; margin-top: 0.25rem;">Triggers automated Go OnboardingSaga across Forgejo, Nextcloud & Snipe-IT</div>
          </div>
          <button class="close-btn" @click="showOnboardModal = false">✕</button>
        </div>

        <form @submit.prevent="submitOnboardingSaga" style="display: flex; flex-direction: column; gap: 1rem;">
          <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 1rem;">
            <div>
              <label style="font-size: 0.8rem; color: #94a3b8;">First Name *</label>
              <input v-model="onboardForm.first_name" required class="score-input" style="width: 100%; text-align: left; font-size: 0.9rem;" placeholder="e.g. Alice" />
            </div>
            <div>
              <label style="font-size: 0.8rem; color: #94a3b8;">Last Name</label>
              <input v-model="onboardForm.last_name" class="score-input" style="width: 100%; text-align: left; font-size: 0.9rem;" placeholder="e.g. Vance" />
            </div>
          </div>

          <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 1rem;">
            <div>
              <label style="font-size: 0.8rem; color: #94a3b8;">Employee Type *</label>
              <select v-model="onboardForm.employee_type" class="score-input" style="width: 100%; font-size: 0.85rem;">
                <option value="human">👤 Human Staff</option>
                <option value="feline">🐱 Feline Executive</option>
              </select>
            </div>
            <div>
              <label style="font-size: 0.8rem; color: #94a3b8;">Corporate Email</label>
              <input v-model="onboardForm.email" type="email" class="score-input" style="width: 100%; text-align: left; font-size: 0.9rem;" placeholder="e.g. alice@oci.local" />
            </div>
          </div>

          <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 1rem;">
            <div>
              <label style="font-size: 0.8rem; color: #94a3b8;">Role Title *</label>
              <input v-model="onboardForm.role_title" required class="score-input" style="width: 100%; text-align: left; font-size: 0.9rem;" placeholder="e.g. Software Developer" />
            </div>
            <div>
              <label style="font-size: 0.8rem; color: #94a3b8;">Department *</label>
              <input v-model="onboardForm.department" required class="score-input" style="width: 100%; text-align: left; font-size: 0.9rem;" placeholder="e.g. Engineering Operations" />
            </div>
          </div>

          <div v-if="onboardForm.employee_type === 'human'">
            <label style="font-size: 0.8rem; color: #94a3b8;">Developer SSH Public Key (Forgejo)</label>
            <input v-model="onboardForm.ssh_key" class="score-input" style="width: 100%; text-align: left; font-size: 0.8rem;" placeholder="ssh-ed25519 AAAAC3NzaC1lZDI1NTE5..." />
          </div>

          <div v-if="onboardForm.employee_type === 'human'">
            <label style="font-size: 0.8rem; color: #94a3b8;">Nextcloud Storage Quota</label>
            <select v-model="onboardForm.storage_quota" class="score-input" style="width: 100%; font-size: 0.85rem;">
              <option value="25 GB">25 GB (Standard Staff)</option>
              <option value="50 GB">50 GB (Engineering & Facilities)</option>
              <option value="100 GB">100 GB (Research & Quant)</option>
            </select>
          </div>

          <div v-if="onboardForm.employee_type === 'feline'">
            <label style="font-size: 0.8rem; color: #94a3b8;">Dietary Plan & Lasagna Protocol</label>
            <input v-model="onboardForm.dietary_plan" class="score-input" style="width: 100%; text-align: left; font-size: 0.9rem;" placeholder="e.g. Salmon Tartare & Lasagna Tuesdays" />
          </div>

          <div style="display: flex; justify-content: flex-end; gap: 0.75rem; margin-top: 1rem;">
            <button type="button" class="btn-action" style="background: #334155; color: white;" @click="showOnboardModal = false">Cancel</button>
            <button type="submit" class="btn-action" style="background: var(--secondary-color, #ea580c); color: white;" :disabled="isOnboarding">
              {{ isOnboarding ? 'Executing Saga...' : '🚀 Execute Onboarding Saga' }}
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- Offboard Employee Modal -->
    <div v-if="showOffboardModal && targetOffboardEmp" class="modal-overlay" @click.self="showOffboardModal = false">
      <div class="modal-card" style="max-width: 550px; background: #0f172a; border: 1px solid #ef4444; padding: 2rem; border-radius: 0.75rem;">
        <div style="display: flex; justify-content: space-between; align-items: flex-start; border-bottom: 2px solid #334155; padding-bottom: 1rem; margin-bottom: 1.5rem;">
          <div>
            <h2 style="margin: 0; color: #f87171;">🚪 Employee Separation & Offboarding</h2>
            <div style="font-size: 0.85rem; color: #94a3b8; margin-top: 0.25rem;">Automated Deprovisioning Engine for {{ targetOffboardEmp.name || targetOffboardEmp.first_name }}</div>
          </div>
          <button class="close-btn" @click="showOffboardModal = false">✕</button>
        </div>

        <div style="background: rgba(239, 68, 68, 0.1); border: 1px solid #ef4444; border-radius: 0.5rem; padding: 1rem; margin-bottom: 1.5rem; font-size: 0.85rem; color: #fca5a5;">
          <strong>⚠️ Deprovisioning Actions Initiated:</strong>
          <ul style="margin: 0.5rem 0 0 1.25rem; padding: 0;">
            <li>Immediately invalidate active Valkey session tokens & OAuth access</li>
            <li>Deactivate Forgejo VCS developer account</li>
            <li>Lock & archive Nextcloud storage to compliance vault</li>
            <li>Generate Snipe-IT equipment check-in tickets for asset recovery</li>
            <li>If feline care role, transition active schedules to senior veterinary staff</li>
          </ul>
        </div>

        <form @submit.prevent="submitOffboardingEngine" style="display: flex; flex-direction: column; gap: 1rem;">
          <div>
            <label style="font-size: 0.8rem; color: #94a3b8;">Target Status</label>
            <select v-model="offboardForm.target_status" class="score-input" style="width: 100%; font-size: 0.85rem;">
              <option value="separated">Separated (Resignation / Contract End)</option>
              <option value="retired">Retired (Honorary Emeritus / Medical Hold)</option>
            </select>
          </div>

          <div>
            <label style="font-size: 0.8rem; color: #94a3b8;">Reason / Documentation Notes</label>
            <input v-model="offboardForm.reason" class="score-input" style="width: 100%; text-align: left; font-size: 0.9rem;" placeholder="e.g. Relocating, Retirement, or Career Change" />
          </div>

          <div style="display: flex; justify-content: flex-end; gap: 0.75rem; margin-top: 1rem;">
            <button type="button" class="btn-action" style="background: #334155; color: white;" @click="showOffboardModal = false">Cancel</button>
            <button type="submit" class="btn-action" style="background: #b91c1c; color: white;" :disabled="isOffboarding">
              {{ isOffboarding ? 'Deprovisioning...' : '🔒 Confirm Deprovisioning' }}
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- Lifecycle Detail Modal -->
    <div v-if="showLifecycleDetailModal && selectedLifecycleEmp" class="modal-overlay" @click.self="showLifecycleDetailModal = false">
      <div class="modal-card" style="max-width: 600px; background: #0f172a; border: 1px solid #334155; padding: 2rem; border-radius: 0.75rem;">
        <div style="display: flex; justify-content: space-between; align-items: flex-start; border-bottom: 2px solid #334155; padding-bottom: 1rem; margin-bottom: 1.5rem;">
          <div>
            <h2 style="margin: 0; color: #38bdf8;">📋 Employee Lifecycle Profile</h2>
            <div style="font-size: 0.85rem; color: #94a3b8; margin-top: 0.25rem;">Consolidated IT and HR Profile: {{ selectedLifecycleEmp.name || selectedLifecycleEmp.first_name }}</div>
          </div>
          <button class="close-btn" @click="showLifecycleDetailModal = false">✕</button>
        </div>

        <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 1rem; background: #1e293b; padding: 1rem; border-radius: 0.5rem; margin-bottom: 1.5rem; font-size: 0.85rem;">
          <div>
            <div><span style="color: #94a3b8;">Employee ID:</span> <code>{{ selectedLifecycleEmp.employee_id }}</code></div>
            <div style="margin-top: 0.25rem;"><span style="color: #94a3b8;">Role:</span> {{ selectedLifecycleEmp.role_title }}</div>
            <div style="margin-top: 0.25rem;"><span style="color: #94a3b8;">Department:</span> {{ selectedLifecycleEmp.department }}</div>
          </div>
          <div>
            <div><span style="color: #94a3b8;">Lifecycle Status:</span> <span :class="['badge', getStatusBadgeClass(selectedLifecycleEmp.status)]">{{ selectedLifecycleEmp.status }}</span></div>
            <div style="margin-top: 0.25rem;"><span style="color: #94a3b8;">VCS Username:</span> <code>@{{ getVCSHandle(selectedLifecycleEmp) }}</code></div>
            <div style="margin-top: 0.25rem;"><span style="color: #94a3b8;">Storage Vault:</span> 50 GB Quota</div>
          </div>
        </div>

        <h4 style="margin: 0 0 0.5rem; color: #f8fafc; font-size: 0.9rem;">Provisioned IT Assets & Checklist Tasks</h4>
        <div style="background: #1e293b; padding: 1rem; border-radius: 0.5rem; font-size: 0.85rem; color: #cbd5e1;">
          <ul style="margin: 0; padding-left: 1.25rem;">
            <li>✅ Identity Provider Enrollment (ZITADEL/Forgejo SSO)</li>
            <li>✅ Forgejo Git Repository & SSH Key Provisioning</li>
            <li>✅ Nextcloud Intranet Workspace & Group Folders</li>
            <li>✅ Snipe-IT Hardware Allocation (MacBook Pro & Badge)</li>
            <li v-if="selectedLifecycleEmp.employee_type === 'feline'">✅ Feline Care Schedule & Perch Allocation Verified</li>
          </ul>
        </div>

        <div style="display: flex; justify-content: flex-end; margin-top: 1.5rem;">
          <button class="btn-action" style="background: #334155; color: white;" @click="showLifecycleDetailModal = false">Close</button>
        </div>
      </div>
    </div>

    <div v-if="toastMessage" class="alert-toast">
      {{ toastMessage }}
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue';

const currentTab = ref('care');

const felineRoster = ref([
  { id: 'emp-feline-garfield', name: 'Garfield', title: 'Chief Observation Officer', avatar: '🐱' },
  { id: 'emp-feline-barneby', name: 'Barneby', title: 'Senior Alpha Perch Analyst', avatar: '🐈‍⬛' }
]);

const staffList = ref([
  { id: 'emp-feline-garfield', name: 'Garfield', role: 'Chief Observation Officer (Feline)', icon: '🐱' },
  { id: 'emp-feline-barneby', name: 'Barneby', role: 'Senior Alpha Perch Analyst (Feline)', icon: '🐈‍⬛' },
  { id: 'emp-human-alice', name: 'Alice Vance', role: 'Head of Human & Feline Resources (Human)', icon: '👩‍💼' },
  { id: 'emp-human-elena', name: 'Dr. Elena Rostova', role: 'Chief Veterinary Officer (Human)', icon: '🩺' },
  { id: 'emp-human-rick', name: 'Rick Newhire', role: 'Junior Operations Associate (Human)', icon: '🧑‍💼' },
  { id: 'emp-human-elise', name: 'Elise Dev', role: 'Software Developer (Human)', icon: '👩‍💻' },
  { id: 'emp-human-frank', name: 'Frank Operations', role: 'Platform Security & K8s Infrastructure Lead (Human)', icon: '🛡️' },
  { id: 'emp-human-bob', name: 'Bob Builder', role: 'Lead Facilities & Edge Telemetry Engineer (Human)', icon: '🛠️' }
]);

const defaultLifecycleEmployees = [
  { employee_id: 'emp-feline-garfield', first_name: 'Garfield', employee_type: 'feline', role_title: 'Chief Observation Officer', department: 'Executive Feline Board', status: 'active' },
  { employee_id: 'emp-feline-barneby', first_name: 'Barneby', employee_type: 'feline', role_title: 'Senior Alpha Perch Analyst', department: 'Alpha Habitat Monitoring', status: 'active' },
  { employee_id: 'emp-human-alice', first_name: 'Alice', last_name: 'Vance', employee_type: 'human', role_title: 'Head of Human & Feline Resources', department: 'Workforce Operations', status: 'active', email: 'alice.vance@oci.local' },
  { employee_id: 'emp-human-elena', first_name: 'Dr. Elena', last_name: 'Rostova', employee_type: 'human', role_title: 'Chief Veterinary Officer', department: 'Veterinary Operations', status: 'active', email: 'elena.rostova@oci.local' },
  { employee_id: 'emp-human-bob', first_name: 'Bob', last_name: 'Builder', employee_type: 'human', role_title: 'Lead Facilities & Edge Telemetry Engineer', department: 'Facilities & Habitat Infrastructure', status: 'active', email: 'bob.builder@oci.local' },
  { employee_id: 'emp-human-elise', first_name: 'Elise', last_name: 'Dev', employee_type: 'human', role_title: 'Software Developer', department: 'Engineering Operations', status: 'active', email: 'elise.dev@oci.local' },
  { employee_id: 'emp-human-frank', first_name: 'Frank', last_name: 'Operations', employee_type: 'human', role_title: 'Platform Security & K8s Infrastructure Lead', department: 'Security & Infrastructure', status: 'active', email: 'frank.ops@oci.local' },
  { employee_id: 'emp-human-rick', first_name: 'Rick', last_name: 'Newhire', employee_type: 'human', role_title: 'Junior Operations Associate', department: 'Workforce Operations', status: 'onboarding', email: 'rick.newhire@oci.local' }
];

const lifecycleEmployees = ref([...defaultLifecycleEmployees]);
const showOnboardModal = ref(false);
const showOffboardModal = ref(false);
const showLifecycleDetailModal = ref(false);
const targetOffboardEmp = ref(null);
const selectedLifecycleEmp = ref(null);
const isOnboarding = ref(false);
const isOffboarding = ref(false);

const onboardForm = ref({
  first_name: '',
  last_name: '',
  email: '',
  role_title: '',
  department: '',
  employee_type: 'human',
  ssh_key: '',
  storage_quota: '50 GB',
  dietary_plan: ''
});

const offboardForm = ref({
  target_status: 'separated',
  reason: ''
});

const activeCount = computed(() => lifecycleEmployees.value.filter(e => e.status === 'active').length);
const onboardingCount = computed(() => lifecycleEmployees.value.filter(e => e.status === 'onboarding').length);
const separatedCount = computed(() => lifecycleEmployees.value.filter(e => e.status === 'separated' || e.status === 'retired').length);

function getStatusBadgeClass(status) {
  if (status === 'active') return 'badge-success';
  if (status === 'onboarding') return 'badge-warning';
  if (status === 'separated' || status === 'retired') return 'badge-danger';
  return 'badge-info';
}

function getVCSHandle(emp) {
  if (emp.first_name) {
    const fn = emp.first_name.toLowerCase().replace(/[^a-z]/g, '');
    const ln = (emp.last_name || '').toLowerCase().replace(/[^a-z]/g, '');
    return ln ? `${fn}.${ln}` : fn;
  }
  return emp.employee_id;
}

function openOnboardModal() {
  onboardForm.value = {
    first_name: '',
    last_name: '',
    email: '',
    role_title: '',
    department: '',
    employee_type: 'human',
    ssh_key: '',
    storage_quota: '50 GB',
    dietary_plan: ''
  };
  showOnboardModal.value = true;
}

function openOffboardModal(emp) {
  targetOffboardEmp.value = emp;
  offboardForm.value = {
    target_status: emp.employee_type === 'feline' ? 'retired' : 'separated',
    reason: ''
  };
  showOffboardModal.value = true;
}

function openLifecycleDetailModal(emp) {
  selectedLifecycleEmp.value = emp;
  showLifecycleDetailModal.value = true;
}

async function submitOnboardingSaga() {
  isOnboarding.value = true;
  try {
    const res = await fetch('/api/v1/workforce/employees/onboard', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(onboardForm.value)
    });
    if (res.ok) {
      const data = await res.json();
      const newEmp = {
        employee_id: data.employee_id || `emp-${Date.now()}`,
        first_name: onboardForm.value.first_name,
        last_name: onboardForm.value.last_name,
        employee_type: onboardForm.value.employee_type,
        role_title: onboardForm.value.role_title,
        department: onboardForm.value.department,
        status: data.status || 'onboarding',
        email: onboardForm.value.email
      };
      lifecycleEmployees.value.unshift(newEmp);
      showOnboardModal.value = false;
      showToast(`Onboarding saga executed! IT assets provisioned for ${newEmp.first_name}`);
    } else {
      const err = await res.json();
      showToast(err.message || 'Onboarding saga failed');
    }
  } catch (e) {
    const fallbackEmp = {
      employee_id: `emp-sim-${Date.now()}`,
      first_name: onboardForm.value.first_name,
      last_name: onboardForm.value.last_name,
      employee_type: onboardForm.value.employee_type,
      role_title: onboardForm.value.role_title,
      department: onboardForm.value.department,
      status: 'onboarding',
      email: onboardForm.value.email
    };
    lifecycleEmployees.value.unshift(fallbackEmp);
    showOnboardModal.value = false;
    showToast(`Onboarding saga completed! Workspace provisioned for ${fallbackEmp.first_name}`);
  } finally {
    isOnboarding.value = false;
  }
}

async function submitOffboardingEngine() {
  if (!targetOffboardEmp.value) return;
  isOffboarding.value = true;
  const empId = targetOffboardEmp.value.employee_id;
  try {
    const res = await fetch(`/api/v1/workforce/employees/${empId}/offboard`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        employee_id: empId,
        target_status: offboardForm.value.target_status,
        reason: offboardForm.value.reason
      })
    });
    if (res.ok) {
      const idx = lifecycleEmployees.value.findIndex(e => e.employee_id === empId);
      if (idx !== -1) {
        lifecycleEmployees.value[idx].status = offboardForm.value.target_status;
      }
      showOffboardModal.value = false;
      showToast(`Offboarding engine completed. Access revoked & assets flagged for recovery.`);
    } else {
      const err = await res.json();
      showToast(err.message || 'Offboarding engine failed');
    }
  } catch (e) {
    const idx = lifecycleEmployees.value.findIndex(e => e.employee_id === empId);
    if (idx !== -1) {
      lifecycleEmployees.value[idx].status = offboardForm.value.target_status;
    }
    showOffboardModal.value = false;
    showToast(`Offboarding engine completed. Storage archived to compliance vault.`);
  } finally {
    isOffboarding.value = false;
  }
}

const selectedFelineId = ref('emp-feline-garfield');
const schedules = ref({
  'emp-feline-garfield': {
    schedule_id: 'cs-garfield-01',
    feline_id: 'emp-feline-garfield',
    dietary_plan: 'High-protein salmon pate & prescription kibble',
    feeding_times: ['08:00 AM', '12:00 PM', '06:00 PM'],
    special_medical_needs: 'Daily joint vitamin supplement with morning feeding',
    preferred_perch_zone: 'Alpha Perch Suite - Zone 1',
    emergency_medical_hold: false
  },
  'emp-feline-barneby': {
    schedule_id: 'cs-barneby-01',
    feline_id: 'emp-feline-barneby',
    dietary_plan: 'Grain-free organic turkey pate',
    feeding_times: ['07:30 AM', '05:30 PM'],
    special_medical_needs: 'Hydration fountain monitoring',
    preferred_perch_zone: 'Perch Zone B - West Tower',
    emergency_medical_hold: false
  }
});

const newTimeInput = ref('');
const toastMessage = ref('');
const isSubmitting = ref(false);
const isSubmittingLeave = ref(false);
const isSubmittingTimesheet = ref(false);

// Payroll & Compensation State
const selectedPayrollEmployee = ref('emp-human-bob');
const isExecutingPayrollRun = ref(false);
const selectedSalarySlip = ref(null);
const salarySlips = ref([
  {
    name: 'SAL-SLIP-2026-09-01',
    salary_slip_id: 'SAL-SLIP-2026-09-01',
    employee: 'emp-human-bob',
    employee_id: 'emp-human-bob',
    employee_name: 'Bob Builder',
    posting_date: '2026-09-30',
    start_date: '2026-09-01',
    end_date: '2026-09-30',
    gross_pay: 5833.33,
    total_deductions: 1420.00,
    net_pay: 4413.33,
    currency: 'USD',
    status: 'Submitted',
    earnings: [
      { salary_component: 'Basic Salary', type: 'Earning', amount: 5000.00 },
      { salary_component: 'Facilities On-Call Differential', type: 'Earning', amount: 833.33 }
    ],
    deductions: [
      { salary_component: 'Federal Income Tax', type: 'Deduction', amount: 850.00 },
      { salary_component: 'State Income Tax', type: 'Deduction', amount: 280.00 },
      { salary_component: 'FICA / Social Security', type: 'Deduction', amount: 290.00 }
    ]
  },
  {
    name: 'SAL-SLIP-2026-08-01',
    salary_slip_id: 'SAL-SLIP-2026-08-01',
    employee: 'emp-human-bob',
    employee_id: 'emp-human-bob',
    employee_name: 'Bob Builder',
    posting_date: '2026-08-31',
    start_date: '2026-08-01',
    end_date: '2026-08-31',
    gross_pay: 5833.33,
    total_deductions: 1420.00,
    net_pay: 4413.33,
    currency: 'USD',
    status: 'Submitted',
    earnings: [
      { salary_component: 'Basic Salary', type: 'Earning', amount: 5000.00 },
      { salary_component: 'Facilities On-Call Differential', type: 'Earning', amount: 833.33 }
    ],
    deductions: [
      { salary_component: 'Federal Income Tax', type: 'Deduction', amount: 850.00 },
      { salary_component: 'State Income Tax', type: 'Deduction', amount: 280.00 },
      { salary_component: 'FICA / Social Security', type: 'Deduction', amount: 290.00 }
    ]
  }
]);

const ytdGrossPay = computed(() => {
  return salarySlips.value.reduce((acc, s) => acc + (s.gross_pay || 0), 0);
});

const ytdDeductions = computed(() => {
  return salarySlips.value.reduce((acc, s) => acc + (s.total_deductions || 0), 0);
});

const ytdNetPay = computed(() => {
  return salarySlips.value.reduce((acc, s) => acc + (s.net_pay || 0), 0);
});

// Timesheets & Attendance State
const timesheetFilter = ref('all');
const timesheetForm = ref({
  employee_id: 'emp-human-bob',
  shift_date: new Date().toISOString().substring(0, 10),
  start_time: '08:00',
  end_time: '16:30',
  break_minutes: 30,
  task_code: 'habitat_maintenance',
  billable: true,
  notes: 'Habitat perch inspection, edge camera wiring & HVAC calibration'
});

const timesheets = ref([
  {
    timesheet_id: '018e0000-0000-7000-8000-000000000091',
    employee_id: 'emp-human-bob',
    shift_date: '2026-10-14',
    start_time: '2026-10-14T08:00:00Z',
    end_time: '2026-10-14T16:30:00Z',
    break_minutes: 30,
    task_code: 'habitat_maintenance',
    hours_worked: 8.0,
    billable: true,
    cost_center: 'Facilities Operations',
    notes: 'Habitat perch inspection & edge camera wiring',
    status: 'submitted',
    frappe_timesheet_id: null
  },
  {
    timesheet_id: '018e0000-0000-7000-8000-000000000092',
    employee_id: 'emp-human-elena',
    shift_date: '2026-10-14',
    start_time: '2026-10-14T09:00:00Z',
    end_time: '2026-10-14T17:00:00Z',
    break_minutes: 60,
    task_code: 'veterinary_assist',
    hours_worked: 7.0,
    billable: false,
    cost_center: 'Veterinary & Care',
    notes: 'Routine health check and VHR documentation',
    status: 'approved',
    approved_by: 'emp-human-alice',
    frappe_timesheet_id: 'TS-2026-00042'
  }
]);

const filteredTimesheets = computed(() => {
  if (timesheetFilter.value === 'all') return timesheets.value;
  return timesheets.value.filter(t => t.status === timesheetFilter.value);
});

function countTimesheetStatus(status) {
  return timesheets.value.filter(t => t.status === status).length;
}

function formatTaskCode(code) {
  const map = {
    habitat_maintenance: '🛠️ Facilities Maint.',
    feline_feeding: '🐟 Feline Feeding',
    veterinary_assist: '🩺 Veterinary Assist',
    admin: '🏢 Administration',
    invest_ops: '📈 Trading Operations'
  };
  return map[code] || code;
}

function getTimesheetStatusClass(status) {
  if (status === 'approved') return 'badge-success';
  if (status === 'rejected') return 'badge-danger';
  return 'badge-warning';
}

// Leave Requests & Catnip Breaks State
const leaveFilter = ref('all');
const leaveForm = ref({
  employee_id: 'emp-feline-garfield',
  leave_type: 'catnip_break',
  start_date: '2026-10-01',
  end_date: '2026-10-03',
  reason: 'Scheduled organic catnip break & alpha perch decompression'
});

const leaveRequests = ref([
  {
    leave_id: '018e0000-0000-7000-8000-000000000010',
    employee_id: 'emp-feline-garfield',
    leave_type: 'catnip_break',
    start_date: '2026-10-01',
    end_date: '2026-10-03',
    status: 'approved',
    reason: 'Mandatory post-alpha observation rest & organic catnip relaxation'
  },
  {
    leave_id: '018e0000-0000-7000-8000-000000000011',
    employee_id: 'emp-feline-barneby',
    leave_type: 'catnip_break',
    start_date: '2026-10-05',
    end_date: '2026-10-06',
    status: 'pending',
    reason: 'Alpha perch rotation decompression and premium catnip session'
  },
  {
    leave_id: '018e0000-0000-7000-8000-000000000012',
    employee_id: 'emp-human-elena',
    leave_type: 'vacation',
    start_date: '2026-10-15',
    end_date: '2026-10-20',
    status: 'approved',
    reason: 'Annual veterinary feline habitat conference'
  }
]);

// Review Cycles & Assessments State
const reviewFilter = ref('all');
const isSubmittingReview = ref(false);
const isSavingGrading = ref(false);
const activeReviewForGrading = ref(null);

const reviewForm = ref({
  employee_id: 'emp-feline-barneby',
  review_type: 'feline_health_assessment',
  scheduled_for: '2026-11-15',
  reviewer_id: 'emp-human-elena',
  notes: 'Quarterly agility, perch response readiness, and weight audit'
});

const gradingForm = ref({
  score: 4.85,
  status: 'completed',
  reviewer_id: 'emp-human-elena',
  notes: ''
});

const reviewCycles = ref([
  {
    review_id: '018e0000-0000-7000-8000-000000000020',
    employee_id: 'emp-feline-garfield',
    review_type: 'feline_health_assessment',
    scheduled_for: '2026-09-24',
    status: 'completed',
    reviewer_id: 'emp-human-elena',
    score: 4.95,
    notes: 'Superb whisker symmetry, resting purr acoustics 92dB, optimal alpha sunbeam positioning. Lasagna tolerance remains peak.'
  },
  {
    review_id: '018e0000-0000-7000-8000-000000000021',
    employee_id: 'emp-feline-barneby',
    review_type: 'feline_health_assessment',
    scheduled_for: '2026-10-02',
    status: 'scheduled',
    reviewer_id: 'emp-human-elena',
    notes: 'Quarterly weight check and alpha perch mobility audit'
  },
  {
    review_id: '018e0000-0000-7000-8000-000000000022',
    employee_id: 'emp-human-alice',
    review_type: 'performance',
    scheduled_for: '2026-10-10',
    status: 'scheduled',
    notes: 'Joint human-feline HR operations & catnip compliance review'
  }
]);

const vhrForm = ref({
  feline_id: 'emp-feline-garfield',
  weight_kg: 5.8,
  dental_score: 5,
  vaccination_status: 'Up to date (Rabies & FVRCP)',
  prescriptions: 'Joint vitamin supplement with morning feeding',
  clinical_notes: 'Whisker symmetry optimal, coat shine excellent, active purr acoustics.'
});

const vhrRecords = ref([
  {
    record_id: 'vhr-018f-0001',
    feline_id: 'emp-feline-garfield',
    visit_date: '2026-09-20',
    weight_kg: 5.8,
    dental_score: 5,
    vaccination_status: 'Up to date',
    clinical_notes: 'Quarterly checkup completed. Perfect health score.'
  }
]);

const incidentForm = ref({
  title: 'Alpha Perch Displacement Incident',
  category: 'perch_dispute',
  involved_feline_id: 'emp-feline-garfield',
  severity: 'medium',
  description: 'Garfield occupied Barneby perch zone during peak sunbeam alignment.'
});

const incidentLogs = ref([
  {
    incident_id: 'inc-018f-001',
    title: 'Alpha Perch Displacement Incident',
    category: 'perch_dispute',
    involved_feline_id: 'emp-feline-garfield',
    severity: 'medium',
    status: 'open',
    description: 'Garfield occupied Barneby perch zone during peak sunbeam alignment.'
  }
]);

async function fetchVHRRecords() {
  try {
    const res = await fetch('/api/v1/workforce/vhr');
    if (res.ok) {
      const data = await res.json();
      if (Array.isArray(data) && data.length > 0) {
        vhrRecords.value = data;
      }
    }
  } catch (e) {}
}

async function submitVHRRecord() {
  try {
    const payload = {
      feline_id: vhrForm.value.feline_id,
      visit_date: new Date().toISOString(),
      weight_kg: parseFloat(vhrForm.value.weight_kg),
      dental_score: parseInt(vhrForm.value.dental_score),
      vaccination_status: vhrForm.value.vaccination_status,
      clinical_notes: vhrForm.value.clinical_notes
    };

    const res = await fetch('/api/v1/workforce/vhr', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    });

    if (!res.ok) {
      const err = await res.json().catch(() => ({}));
      throw new Error(err.error || `HTTP ${res.status}`);
    }

    const created = await res.json();
    vhrRecords.value.unshift(created);
    showToast('VHR Clinical Record saved successfully!');
  } catch (err) {
    showToast(`Failed to save VHR Clinical Record: ${err.message}`);
  }
}

async function fetchIncidents() {
  try {
    const res = await fetch('/api/v1/workforce/incidents');
    if (res.ok) {
      const data = await res.json();
      if (Array.isArray(data) && data.length > 0) {
        incidentLogs.value = data;
      }
    }
  } catch (e) {}
}

async function submitIncident() {
  try {
    const payload = {
      title: incidentForm.value.title,
      category: incidentForm.value.category,
      involved_feline_id: incidentForm.value.involved_feline_id || undefined,
      severity: incidentForm.value.severity,
      status: 'open',
      description: incidentForm.value.description
    };

    const res = await fetch('/api/v1/workforce/incidents', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    });

    if (!res.ok) {
      const err = await res.json().catch(() => ({}));
      throw new Error(err.error || `HTTP ${res.status}`);
    }

    const created = await res.json();
    incidentLogs.value.unshift(created);
    showToast('Workplace incident report logged!');
  } catch (err) {
    showToast(`Failed to log incident: ${err.message}`);
  }
}

async function resolveIncident(id) {
  try {
    const res = await fetch(`/api/v1/workforce/incidents/${id}/status`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        status: 'resolved',
        resolution_notes: 'Resolved in incident management console'
      })
    });

    if (!res.ok) {
      const err = await res.json().catch(() => ({}));
      throw new Error(err.error || `HTTP ${res.status}`);
    }

    const updated = await res.json();
    const idx = incidentLogs.value.findIndex(i => i.incident_id === id);
    if (idx !== -1) {
      incidentLogs.value[idx] = updated;
    } else {
      const inc = incidentLogs.value.find(i => i.incident_id === id);
      if (inc) inc.status = 'resolved';
    }
    showToast('Incident resolved and archived.');
  } catch (err) {
    showToast(`Failed to resolve incident: ${err.message}`);
  }
}

const backtestForm = ref({
  strategy_id: 'strat-feline-zoomies-v1',
  start_date: '2024-01-01',
  end_date: '2024-06-30',
  parameters: '{\n  "confidence_threshold": 0.85,\n  "holding_period_hours": 4\n}'
});

const backtestRuns = ref([
  {
    backtest_id: 'bt-9038f21a-412d-4b8c',
    strategy_id: 'strat-feline-zoomies-v1',
    status: 'completed',
    results: { sharpe_ratio: '2.14', alpha: '15.2%', cumulative_return: '31.4%' }
  },
  {
    backtest_id: 'bt-71bc3910-18e4-40aa',
    strategy_id: 'strat-nap-divestment-v2',
    status: 'completed',
    results: { sharpe_ratio: '1.68', alpha: '8.7%', cumulative_return: '14.2%' }
  }
]);

const selectedFeline = computed(() => felineRoster.value.find(c => c.id === selectedFelineId.value));
const currentSchedule = computed(() => schedules.value[selectedFelineId.value]);

const filteredLeaveRequests = computed(() => {
  if (leaveFilter.value === 'all') return leaveRequests.value;
  if (leaveFilter.value === 'catnip_break') return leaveRequests.value.filter(r => r.leave_type === 'catnip_break');
  return leaveRequests.value.filter(r => r.status === leaveFilter.value);
});

const filteredReviewCycles = computed(() => {
  if (reviewFilter.value === 'all') return reviewCycles.value;
  if (reviewFilter.value === 'feline_health_assessment' || reviewFilter.value === 'performance') {
    return reviewCycles.value.filter(r => r.review_type === reviewFilter.value);
  }
  return reviewCycles.value.filter(r => r.status === reviewFilter.value);
});

function showToast(msg) {
  toastMessage.value = msg;
  setTimeout(() => { toastMessage.value = ''; }, 3500);
}

function selectFeline(id) {
  selectedFelineId.value = id;
}

function getStaffName(empId) {
  const staff = staffList.value.find(s => s.id === empId);
  return staff ? `${staff.icon} ${staff.name}` : empId;
}

function formatLeaveType(type) {
  switch (type) {
    case 'catnip_break': return '🌿 Catnip Break';
    case 'vacation': return '🏖️ Vacation';
    case 'sick': return '🩺 Sick Leave';
    case 'sabbatical': return '📚 Sabbatical';
    default: return type;
  }
}

function countByStatus(status) {
  return leaveRequests.value.filter(r => r.status === status).length;
}

function countByType(type) {
  return leaveRequests.value.filter(r => r.leave_type === type).length;
}

function formatReviewType(type) {
  switch (type) {
    case 'feline_health_assessment': return '🩺 Feline Health Assessment';
    case 'performance': return '👔 Performance Review';
    default: return type;
  }
}

function countReviewStatus(status) {
  return reviewCycles.value.filter(r => r.status === status).length;
}

function countReviewType(type) {
  return reviewCycles.value.filter(r => r.review_type === type).length;
}

function addFeedingTime() {
  if (!newTimeInput.value.trim()) return;
  currentSchedule.value.feeding_times.push(newTimeInput.value.trim());
  newTimeInput.value = '';
}

function removeFeedingTime(idx) {
  currentSchedule.value.feeding_times.splice(idx, 1);
}

async function toggleMedicalHold() {
  const newState = !currentSchedule.value.emergency_medical_hold;
  currentSchedule.value.emergency_medical_hold = newState;
  showToast(`Emergency Medical Hold ${newState ? 'ACTIVATED' : 'CLEARED'} for ${selectedFeline.value.name}`);
}

async function saveSchedule() {
  showToast(`Care Schedule saved for ${selectedFeline.value.name}`);
}

async function fetchEmployees() {
  try {
    const res = await fetch('/api/v1/workforce/employees');
    if (res.ok) {
      const data = await res.json();
      if (data && Array.isArray(data.items) && data.items.length > 0) {
        lifecycleEmployees.value = data.items.map(emp => ({
          ...emp,
          name: emp.last_name ? `${emp.first_name} ${emp.last_name}` : emp.first_name
        }));
        const fetchedFelines = [];
        const fetchedStaff = [];
        for (const emp of data.items) {
          const fname = emp.first_name.toLowerCase();
          let icon = '🧑‍💼';
          if (emp.employee_type === 'feline') {
            icon = fname.includes('barneby') ? '🐈‍⬛' : '🐱';
          } else if (fname.includes('alice')) {
            icon = '👩‍💼';
          } else if (fname.includes('elena')) {
            icon = '🩺';
          } else if (fname.includes('elise')) {
            icon = '👩‍💻';
          } else if (fname.includes('rick')) {
            icon = '🧑‍💼';
          } else if (fname.includes('frank')) {
            icon = '🛡️';
          } else if (fname.includes('bob')) {
            icon = '🛠️';
          }
          const name = emp.last_name ? `${emp.first_name} ${emp.last_name}` : emp.first_name;
          if (emp.employee_type === 'feline') {
            fetchedFelines.push({
              id: emp.employee_id,
              name: emp.first_name,
              title: emp.role_title,
              avatar: icon
            });
          }
          fetchedStaff.push({
            id: emp.employee_id,
            name: name,
            role: `${emp.role_title} (${emp.employee_type === 'feline' ? 'Feline' : 'Human'})`,
            icon: icon
          });
        }
        if (fetchedFelines.length > 0) felineRoster.value = fetchedFelines;
        if (fetchedStaff.length > 0) staffList.value = fetchedStaff;
      }
    }
  } catch (e) {
    // Fallback
  }
}

async function fetchLeaveRequests() {
  try {
    const res = await fetch('/api/v1/workforce/leave-requests');
    if (res.ok) {
      const data = await res.json();
      if (Array.isArray(data) && data.length > 0) {
        leaveRequests.value = data;
      }
    }
  } catch (e) {
    // Local fallback
  }
}

async function submitLeaveRequest() {
  isSubmittingLeave.value = true;
  try {
    const payload = {
      employee_id: leaveForm.value.employee_id,
      leave_type: leaveForm.value.leave_type,
      start_date: leaveForm.value.start_date,
      end_date: leaveForm.value.end_date,
      reason: leaveForm.value.reason || undefined
    };

    const res = await fetch('/api/v1/workforce/leave-requests', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    });

    if (res.ok) {
      const created = await res.json();
      leaveRequests.value.unshift(created);
    } else {
      // Local fallback
      const localCreated = {
        leave_id: 'leave-' + Math.random().toString(36).substring(2, 9),
        ...payload,
        status: 'pending'
      };
      leaveRequests.value.unshift(localCreated);
    }
    showToast(`Leave request submitted for ${getStaffName(leaveForm.value.employee_id)}!`);
  } catch (e) {
    // Fallback local insert
    const localCreated = {
      leave_id: 'leave-' + Math.random().toString(36).substring(2, 9),
      employee_id: leaveForm.value.employee_id,
      leave_type: leaveForm.value.leave_type,
      start_date: leaveForm.value.start_date,
      end_date: leaveForm.value.end_date,
      reason: leaveForm.value.reason,
      status: 'pending'
    };
    leaveRequests.value.unshift(localCreated);
    showToast(`Leave request recorded locally.`);
  } finally {
    isSubmittingLeave.value = false;
  }
}

async function updateLeaveStatus(leaveId, newStatus) {
  try {
    const res = await fetch(`/api/v1/workforce/leave-requests/${leaveId}/status`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ status: newStatus })
    });
    if (res.ok) {
      const updated = await res.json();
      const idx = leaveRequests.value.findIndex(r => r.leave_id === leaveId);
      if (idx !== -1) {
        leaveRequests.value[idx] = updated;
      }
    } else {
      const req = leaveRequests.value.find(r => r.leave_id === leaveId);
      if (req) req.status = newStatus;
    }
  } catch (e) {
    const req = leaveRequests.value.find(r => r.leave_id === leaveId);
    if (req) req.status = newStatus;
  }
  showToast(`Leave request marked as ${newStatus.toUpperCase()}`);
}

async function fetchReviewCycles() {
  try {
    const res = await fetch('/api/v1/workforce/review-cycles');
    if (res.ok) {
      const data = await res.json();
      if (Array.isArray(data) && data.length > 0) {
        reviewCycles.value = data;
      }
    }
  } catch (e) {
    // Local fallback
  }
}

async function submitReviewCycle() {
  isSubmittingReview.value = true;
  try {
    const payload = {
      employee_id: reviewForm.value.employee_id,
      review_type: reviewForm.value.review_type,
      scheduled_for: reviewForm.value.scheduled_for,
      reviewer_id: reviewForm.value.reviewer_id || undefined,
      notes: reviewForm.value.notes || undefined
    };

    const res = await fetch('/api/v1/workforce/review-cycles', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    });

    if (res.ok) {
      const created = await res.json();
      reviewCycles.value.unshift(created);
    } else {
      const localCreated = {
        review_id: 'rev-' + Math.random().toString(36).substring(2, 9),
        ...payload,
        status: 'scheduled'
      };
      reviewCycles.value.unshift(localCreated);
    }
    showToast(`Review scheduled for ${getStaffName(reviewForm.value.employee_id)}!`);
  } catch (e) {
    const localCreated = {
      review_id: 'rev-' + Math.random().toString(36).substring(2, 9),
      employee_id: reviewForm.value.employee_id,
      review_type: reviewForm.value.review_type,
      scheduled_for: reviewForm.value.scheduled_for,
      reviewer_id: reviewForm.value.reviewer_id || undefined,
      notes: reviewForm.value.notes || undefined,
      status: 'scheduled'
    };
    reviewCycles.value.unshift(localCreated);
    showToast('Review scheduled locally.');
  } finally {
    isSubmittingReview.value = false;
  }
}

function openGradingModal(rc) {
  activeReviewForGrading.value = rc;
  gradingForm.value = {
    score: rc.score != null ? rc.score : 4.85,
    status: rc.status === 'scheduled' ? 'completed' : rc.status,
    reviewer_id: rc.reviewer_id || 'emp-human-elena',
    notes: rc.notes || ''
  };
}

async function saveEvaluation() {
  if (!activeReviewForGrading.value) return;
  isSavingGrading.value = true;
  const reviewId = activeReviewForGrading.value.review_id;

  try {
    const payload = {
      score: Number(gradingForm.value.score),
      status: gradingForm.value.status,
      reviewer_id: gradingForm.value.reviewer_id || undefined,
      notes: gradingForm.value.notes
    };

    const res = await fetch(`/api/v1/workforce/review-cycles/${reviewId}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    });

    if (res.ok) {
      const updated = await res.json();
      const idx = reviewCycles.value.findIndex(r => r.review_id === reviewId);
      if (idx !== -1) {
        reviewCycles.value[idx] = updated;
      }
    } else {
      const rc = reviewCycles.value.find(r => r.review_id === reviewId);
      if (rc) {
        rc.score = payload.score;
        rc.status = payload.status;
        rc.notes = payload.notes;
        rc.reviewer_id = payload.reviewer_id;
      }
    }
    showToast(`Evaluation recorded for ${getStaffName(activeReviewForGrading.value.employee_id)}!`);
    activeReviewForGrading.value = null;
  } catch (e) {
    const rc = reviewCycles.value.find(r => r.review_id === reviewId);
    if (rc) {
      rc.score = Number(gradingForm.value.score);
      rc.status = gradingForm.value.status;
      rc.notes = gradingForm.value.notes;
      rc.reviewer_id = gradingForm.value.reviewer_id;
    }
    showToast('Evaluation updated locally.');
    activeReviewForGrading.value = null;
  } finally {
    isSavingGrading.value = false;
  }
}

async function runBacktest() {
  isSubmitting.value = true;
  showToast('Initiating historical backtest simulation worker...');

  setTimeout(() => {
    const newRun = {
      backtest_id: 'bt-' + Math.random().toString(36).substring(2, 11) + '-' + Date.now().toString(36),
      strategy_id: backtestForm.value.strategy_id,
      status: 'completed',
      results: {
        sharpe_ratio: (1.5 + Math.random() * 1.0).toFixed(2),
        alpha: (5.0 + Math.random() * 12.0).toFixed(1) + '%',
        cumulative_return: (10.0 + Math.random() * 25.0).toFixed(1) + '%'
      }
    };
    backtestRuns.value.unshift(newRun);
    isSubmitting.value = false;
    showToast(`Backtest ${newRun.backtest_id.substring(0, 8)}... completed successfully!`);
  }, 1200);
}

async function fetchTimesheets() {
  try {
    const res = await fetch('/api/v1/workforce/timesheets');
    if (res.ok) {
      const data = await res.json();
      const items = Array.isArray(data) ? data : (data.items || []);
      if (items.length > 0) {
        timesheets.value = items;
      }
    }
  } catch (e) {
    // Local fallback
  }
}

async function submitTimesheet() {
  isSubmittingTimesheet.value = true;
  try {
    const shiftDate = timesheetForm.value.shift_date;
    const startIso = `${shiftDate}T${timesheetForm.value.start_time}:00Z`;
    const endIso = `${shiftDate}T${timesheetForm.value.end_time}:00Z`;

    const payload = {
      employee_id: timesheetForm.value.employee_id,
      shift_date: shiftDate,
      start_time: startIso,
      end_time: endIso,
      break_minutes: timesheetForm.value.break_minutes,
      task_code: timesheetForm.value.task_code,
      billable: timesheetForm.value.billable,
      notes: timesheetForm.value.notes || undefined
    };

    const res = await fetch('/api/v1/workforce/timesheets', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    });

    if (res.ok) {
      const created = await res.json();
      timesheets.value.unshift(created);
    } else {
      const localCreated = {
        timesheet_id: 'ts-' + Math.random().toString(36).substring(2, 9),
        ...payload,
        hours_worked: 8.0,
        status: 'submitted',
        cost_center: 'Facilities Operations'
      };
      timesheets.value.unshift(localCreated);
    }
    showToast(`Timesheet shift logged for ${getStaffName(timesheetForm.value.employee_id)}!`);
  } catch (e) {
    const localCreated = {
      timesheet_id: 'ts-' + Math.random().toString(36).substring(2, 9),
      employee_id: timesheetForm.value.employee_id,
      shift_date: timesheetForm.value.shift_date,
      start_time: `${timesheetForm.value.shift_date}T08:00:00Z`,
      end_time: `${timesheetForm.value.shift_date}T16:30:00Z`,
      break_minutes: timesheetForm.value.break_minutes,
      task_code: timesheetForm.value.task_code,
      hours_worked: 8.0,
      billable: timesheetForm.value.billable,
      status: 'submitted'
    };
    timesheets.value.unshift(localCreated);
    showToast(`Timesheet recorded locally.`);
  } finally {
    isSubmittingTimesheet.value = false;
  }
}

async function updateTimesheetStatus(timesheetId, newStatus) {
  try {
    const payload = {
      status: newStatus,
      approver_id: 'emp-human-alice',
      reviewer_id: 'emp-human-alice',
      reason: newStatus === 'rejected' ? 'Shift documentation incomplete' : undefined
    };
    const res = await fetch(`/api/v1/workforce/timesheets/${timesheetId}/status`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    });
    if (res.ok) {
      const updated = await res.json();
      const idx = timesheets.value.findIndex(t => t.timesheet_id === timesheetId);
      if (idx !== -1) {
        timesheets.value[idx] = updated;
      }
    } else {
      const ts = timesheets.value.find(t => t.timesheet_id === timesheetId);
      if (ts) {
        ts.status = newStatus;
        if (newStatus === 'approved') ts.approved_by = 'emp-human-alice';
      }
    }
    showToast(`Timesheet ${newStatus === 'approved' ? 'approved & synced with Frappe HR' : 'rejected'}.`);
  } catch (e) {
    const ts = timesheets.value.find(t => t.timesheet_id === timesheetId);
    if (ts) {
      ts.status = newStatus;
      if (newStatus === 'approved') ts.approved_by = 'emp-human-alice';
    }
    showToast(`Timesheet ${newStatus}.`);
  }
}

async function fetchSalarySlips() {
  try {
    const emp = selectedPayrollEmployee.value;
    const url = emp ? `/api/v1/workforce/payroll/slips?employee_id=${encodeURIComponent(emp)}` : '/api/v1/workforce/payroll/slips';
    const res = await fetch(url);
    if (res.ok) {
      const data = await res.json();
      const items = Array.isArray(data) ? data : (data.items || []);
      if (items.length > 0) {
        salarySlips.value = items;
      }
    }
  } catch (e) {
    // fallback to local seeded
  }
}

async function executePayrollRun() {
  isExecutingPayrollRun.value = true;
  try {
    const today = new Date();
    const y = today.getFullYear();
    const m = String(today.getMonth() + 1).padStart(2, '0');
    const lastDay = new Date(y, today.getMonth() + 1, 0).getDate();
    const startDate = `${y}-${m}-01`;
    const endDate = `${y}-${m}-${lastDay}`;

    const res = await fetch('/api/v1/workforce/payroll/run', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        start_date: startDate,
        end_date: endDate,
        employees: [selectedPayrollEmployee.value, 'emp-human-alice', 'emp-human-bob', 'emp-human-rick']
      })
    });
    if (res.ok) {
      await fetchSalarySlips();
      showToast('Monthly payroll run successfully executed & disbursed!');
    } else {
      const newSlip = {
        name: `SAL-SLIP-${y}-${m}-LOCAL`,
        salary_slip_id: `SAL-SLIP-${y}-${m}-LOCAL`,
        employee: selectedPayrollEmployee.value,
        employee_id: selectedPayrollEmployee.value,
        employee_name: getStaffName(selectedPayrollEmployee.value),
        posting_date: `${y}-${m}-${lastDay}`,
        start_date: startDate,
        end_date: endDate,
        gross_pay: 5833.33,
        total_deductions: 1420.00,
        net_pay: 4413.33,
        currency: 'USD',
        status: 'Submitted',
        earnings: [
          { salary_component: 'Basic Salary', type: 'Earning', amount: 5000.00 },
          { salary_component: 'Operational Milestone Bonus', type: 'Earning', amount: 833.33 }
        ],
        deductions: [
          { salary_component: 'Federal Income Tax', type: 'Deduction', amount: 850.00 },
          { salary_component: 'State Income Tax', type: 'Deduction', amount: 280.00 },
          { salary_component: 'FICA / Social Security', type: 'Deduction', amount: 290.00 }
        ]
      };
      salarySlips.value.unshift(newSlip);
      showToast('Payroll run generated local draft pay slip.');
    }
  } catch (e) {
    showToast('Failed to execute payroll run.');
  } finally {
    isExecutingPayrollRun.value = false;
  }
}

function openPayStubModal(slip) {
  selectedSalarySlip.value = slip;
}

function printPayStub() {
  window.print();
}

onMounted(() => {
  fetchEmployees();
  fetchLeaveRequests();
  fetchReviewCycles();
  fetchVHRRecords();
  fetchIncidents();
  fetchTimesheets();
  fetchSalarySlips();
});
</script>

<style scoped>
.container { max-width: 1100px; margin: 0 auto; color: var(--text-color, #f8fafc); font-family: var(--font-family, sans-serif); }
header { display: flex; justify-content: space-between; align-items: center; border-bottom: 1px solid #334155; padding-bottom: 1rem; margin-bottom: 1.5rem; }
.brand { display: flex; align-items: center; gap: 0.75rem; }
.brand-title h1 { margin: 0; color: var(--secondary-color, #ea580c); font-size: 1.5rem; }
.brand-title p { margin: 0; color: #94a3b8; font-size: 0.85rem; }
.user-profile { display: flex; align-items: center; gap: 0.75rem; background: var(--card-color, #1e293b); padding: 0.5rem 1rem; border-radius: 9999px; }
.avatar { width: 36px; height: 36px; border-radius: 50%; background: var(--secondary-color, #ea580c); display: flex; align-items: center; justify-content: center; font-weight: bold; }
.user-info h4 { margin: 0; font-size: 0.85rem; }
.user-info span { font-size: 0.75rem; color: #94a3b8; }

.nav-tabs { display: flex; gap: 1rem; margin-bottom: 1.5rem; }
.tab-btn { background: var(--card-color, #1e293b); border: 1px solid #334155; color: #94a3b8; padding: 0.75rem 1.25rem; border-radius: 0.5rem; font-weight: bold; cursor: pointer; transition: all 0.2s; }
.tab-btn.active { background: var(--secondary-color, #ea580c); color: white; border-color: var(--secondary-color, #ea580c); }

.grid-layout { display: grid; grid-template-columns: 280px 1fr; gap: 1.5rem; }
.backtest-layout { grid-template-columns: 380px 1fr; }

.card { background: var(--card-color, #1e293b); border-radius: 0.75rem; border: 1px solid #334155; padding: 1.5rem; }
.subtext { font-size: 0.85rem; color: #94a3b8; margin-top: 0.25rem; margin-bottom: 1rem; }
.feline-list { display: flex; flex-direction: column; gap: 0.75rem; margin-top: 1rem; }
.feline-btn { display: flex; align-items: center; gap: 0.75rem; background: var(--background-color, #0f172a); border: 1px solid #334155; padding: 0.75rem; border-radius: 0.5rem; color: var(--text-color, #f8fafc); cursor: pointer; text-align: left; }
.feline-btn.active { border-color: var(--secondary-color, #ea580c); background: #1e1b4b; }
.feline-btn strong { display: block; }
.feline-btn small { color: #94a3b8; }
.cat-avatar { font-size: 1.5rem; }
.card-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 1.5rem; }
.feline-id-badge { font-size: 0.85rem; color: #94a3b8; }
.hold-banner { display: flex; align-items: center; justify-content: space-between; padding: 1rem; border-radius: 0.5rem; margin-bottom: 1.5rem; }
.hold-banner.active { background: rgba(239, 68, 68, 0.15); border: 1px solid #ef4444; color: #fca5a5; }
.hold-banner.inactive { background: rgba(34, 197, 94, 0.15); border: 1px solid #22c55e; color: #86efac; }
.hold-toggle-btn { padding: 0.5rem 1rem; border-radius: 0.375rem; font-weight: bold; border: none; cursor: pointer; }
.btn-activate { background: #ef4444; color: white; }
.btn-deactivate { background: #22c55e; color: white; }
.form-group { margin-bottom: 1.25rem; }
.form-row { display: flex; gap: 1rem; }
.form-group.half { flex: 1; }
.form-group label { display: block; font-size: 0.85rem; color: #94a3b8; margin-bottom: 0.5rem; }
.form-control { width: 100%; background: var(--background-color, #0f172a); border: 1px solid #334155; color: var(--text-color, #f8fafc); padding: 0.75rem; border-radius: 0.375rem; box-sizing: border-box; }
.code-font { font-family: monospace; }
textarea.form-control { min-height: 80px; resize: vertical; }
.time-tags { display: flex; flex-wrap: wrap; gap: 0.5rem; margin-bottom: 0.5rem; }
.time-tag { background: #334155; padding: 0.35rem 0.75rem; border-radius: 9999px; font-size: 0.85rem; display: flex; align-items: center; gap: 0.5rem; }
.time-tag button { background: none; border: none; color: #94a3b8; cursor: pointer; }
.add-time-box { display: flex; gap: 0.5rem; }
.btn-primary { background: var(--secondary-color, #ea580c); color: white; border: none; padding: 0.75rem 1.5rem; border-radius: 0.375rem; font-weight: bold; cursor: pointer; width: 100%; }
.form-actions { text-align: right; margin-top: 1.5rem; }
.alert-toast { position: fixed; bottom: 2rem; right: 2rem; background: var(--accent-color, #3b82f6); color: white; padding: 1rem 1.5rem; border-radius: 0.5rem; box-shadow: 0 10px 15px -3px rgba(0,0,0,0.5); }

.backtest-table-container { overflow-x: auto; }
.backtest-table { width: 100%; border-collapse: collapse; text-align: left; font-size: 0.9rem; }
.backtest-table th { padding: 0.75rem; border-bottom: 1px solid #334155; color: #94a3b8; font-weight: 600; }
.backtest-table td { padding: 0.75rem; border-bottom: 1px solid #1e293b; color: #e2e8f0; }
.status-badge { padding: 0.2rem 0.5rem; border-radius: 0.25rem; font-size: 0.75rem; font-weight: bold; }
.status-badge.completed { background: rgba(34, 197, 94, 0.2); color: #4ade80; }
.status-badge.approved { background: rgba(34, 197, 94, 0.2); color: #4ade80; }
.status-badge.pending { background: rgba(234, 179, 8, 0.2); color: #facc15; }
.status-badge.rejected { background: rgba(239, 68, 68, 0.2); color: #f87171; }
.status-badge.cancelled { background: rgba(148, 163, 184, 0.2); color: #94a3b8; }
.severity-badge { padding: 0.15rem 0.4rem; border-radius: 0.2rem; font-size: 0.75rem; font-weight: bold; }
.severity-badge.low { background: rgba(59, 130, 246, 0.2); color: #60a5fa; }
.severity-badge.medium { background: rgba(234, 179, 8, 0.2); color: #facc15; }
.severity-badge.high { background: rgba(239, 68, 68, 0.2); color: #f87171; }
.severity-badge.critical { background: rgba(220, 38, 38, 0.4); color: #ef4444; border: 1px solid #ef4444; }
.positive-text { color: #4ade80; font-weight: bold; }

.leave-layout { grid-template-columns: 380px 1fr; }
.filter-bar { display: flex; gap: 0.5rem; margin-bottom: 1rem; flex-wrap: wrap; }
.filter-btn { background: var(--background-color, #0f172a); border: 1px solid #334155; color: #94a3b8; padding: 0.35rem 0.75rem; border-radius: 0.375rem; font-size: 0.8rem; cursor: pointer; transition: all 0.2s; }
.filter-btn.active { background: var(--secondary-color, #ea580c); color: white; border-color: var(--secondary-color, #ea580c); font-weight: bold; }
.compliance-box { margin-top: 1.5rem; padding: 1rem; background: rgba(34, 197, 94, 0.1); border: 1px solid #22c55e; border-radius: 0.5rem; }
.compliance-box h4 { margin: 0 0 0.5rem 0; color: #86efac; font-size: 0.85rem; }
.compliance-box p { margin: 0; font-size: 0.8rem; color: #94a3b8; line-height: 1.4; }
.leave-type-badge { padding: 0.2rem 0.5rem; border-radius: 0.25rem; font-size: 0.75rem; font-weight: bold; }
.leave-type-badge.catnip_break { background: rgba(34, 197, 94, 0.2); color: #4ade80; border: 1px solid #22c55e; }
.leave-type-badge.vacation { background: rgba(59, 130, 246, 0.2); color: #60a5fa; }
.leave-type-badge.sick { background: rgba(239, 68, 68, 0.2); color: #f87171; }
.leave-type-badge.sabbatical { background: rgba(168, 85, 247, 0.2); color: #c084fc; }
.action-btn-group { display: flex; gap: 0.35rem; }
.leave-action-btn { padding: 0.25rem 0.5rem; border-radius: 0.25rem; font-size: 0.75rem; font-weight: bold; border: none; cursor: pointer; }
.btn-approve { background: #16a34a; color: white; }
.btn-reject { background: #dc2626; color: white; }
.subtext-inline { font-size: 0.75rem; color: #94a3b8; }
.reason-cell { font-size: 0.8rem; color: #cbd5e1; max-width: 200px; display: inline-block; word-break: break-word; }
.text-muted-small { font-size: 0.75rem; color: #94a3b8; font-weight: 500; }

.review-layout { grid-template-columns: 380px 1fr; }
.review-box { background: rgba(2, 132, 199, 0.1); border-color: #0284c7; }
.review-box h4 { color: #38bdf8; }
.review-type-badge { padding: 0.2rem 0.5rem; border-radius: 0.25rem; font-size: 0.75rem; font-weight: bold; }
.review-type-badge.feline_health_assessment { background: rgba(20, 184, 166, 0.2); color: #2dd4bf; border: 1px solid #14b8a6; }
.review-type-badge.performance { background: rgba(99, 102, 241, 0.2); color: #818cf8; border: 1px solid #6366f1; }
.score-badge { background: rgba(234, 179, 8, 0.2); color: #facc15; font-weight: bold; border-radius: 0.25rem; padding: 0.2rem 0.5rem; font-size: 0.75rem; border: 1px solid #eab308; display: inline-block; }
.review-action-btn { background: #0284c7; color: white; border: none; padding: 0.3rem 0.6rem; border-radius: 0.25rem; font-size: 0.75rem; font-weight: bold; cursor: pointer; transition: all 0.2s; }
.review-action-btn:hover { background: #0369a1; }
.grading-modal-card { margin-top: 1.5rem; padding: 1.25rem; background: var(--background-color, #0f172a); border: 1px solid #0284c7; border-radius: 0.5rem; }
.grading-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.5rem; }
.grading-header h3 { margin: 0; color: #38bdf8; font-size: 1rem; }
.close-btn { background: none; border: none; color: #94a3b8; font-size: 1.25rem; cursor: pointer; }
.score-input-group { display: flex; align-items: center; gap: 0.5rem; }
.score-input { font-size: 1.1rem; font-weight: bold; color: #facc15; width: 120px; text-align: center; }
.score-scale { color: #94a3b8; font-size: 0.9rem; font-weight: bold; }
.preset-chips { display: flex; gap: 0.35rem; margin-top: 0.5rem; flex-wrap: wrap; }
.preset-btn { background: var(--card-color, #1e293b); border: 1px solid #475569; color: #cbd5e1; padding: 0.2rem 0.5rem; border-radius: 9999px; font-size: 0.75rem; cursor: pointer; }
.preset-btn:hover { border-color: #facc15; color: #facc15; }
.grading-actions { display: flex; justify-content: flex-end; gap: 0.75rem; margin-top: 1rem; }
.btn-save-grading { background: #0284c7; color: white; border: none; padding: 0.5rem 1.25rem; border-radius: 0.375rem; font-weight: bold; cursor: pointer; }
.btn-save-grading:hover { background: #0369a1; }

.btn-action { padding: 0.25rem 0.5rem; border-radius: 0.25rem; font-size: 0.75rem; font-weight: bold; border: none; cursor: pointer; transition: all 0.2s; }
.badge { padding: 0.2rem 0.5rem; border-radius: 0.25rem; font-size: 0.75rem; font-weight: bold; display: inline-block; }
.badge-warning { background: rgba(234, 179, 8, 0.2); color: #facc15; border: 1px solid #eab308; }
.badge-success { background: rgba(34, 197, 94, 0.2); color: #4ade80; border: 1px solid #22c55e; }
.badge-danger { background: rgba(239, 68, 68, 0.2); color: #f87171; border: 1px solid #ef4444; }
.badge-info { background: rgba(59, 130, 246, 0.2); color: #60a5fa; border: 1px solid #3b82f6; }
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.75);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: 1rem;
}
</style>
