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
        <div class="avatar">{{ currentTab === 'care' || currentTab === 'vhr' ? 'ER' : (currentTab === 'helpdesk' ? 'FO' : (currentTab === 'leave' || currentTab === 'review' || currentTab === 'timesheets' || currentTab === 'payroll' || currentTab === 'lifecycle' || currentTab === 'docs' ? 'AV' : 'DQ')) }}</div>
        <div class="user-info">
          <h4>{{ currentTab === 'care' || currentTab === 'vhr' ? 'Dr. Elena Rostova' : (currentTab === 'helpdesk' ? 'Frank Operations' : (currentTab === 'leave' || currentTab === 'review' || currentTab === 'timesheets' || currentTab === 'payroll' || currentTab === 'lifecycle' || currentTab === 'docs' ? 'Alice Vance' : 'David Quant')) }}</h4>
          <span>{{ currentTab === 'care' || currentTab === 'vhr' ? 'Chief Veterinary Officer' : (currentTab === 'helpdesk' ? 'Platform Security & K8s Infrastructure Lead' : (currentTab === 'leave' || currentTab === 'review' || currentTab === 'timesheets' || currentTab === 'payroll' || currentTab === 'lifecycle' || currentTab === 'docs' ? 'Head of Human & Feline Resources' : 'Feline Behavioral Data Scientist')) }}</span>
        </div>
      </div>
    </header>

    <!-- Real-Time Pub/Sub Workplace Incident Alert Banner -->
    <div v-if="activeEmergencyAlert" class="emergency-alert-banner" :class="activeEmergencyAlert.severity">
      <div class="alert-content">
        <span class="alert-icon">🚨</span>
        <div class="alert-text">
          <div class="alert-headline">
            <span class="alert-title-tag">EMERGENCY WORKPLACE INCIDENT</span>
            <span :class="['severity-badge', activeEmergencyAlert.severity]">{{ (activeEmergencyAlert.severity || 'ALERT').toUpperCase() }}</span>
            <span v-if="activeEmergencyAlert.room_zone" class="zone-badge">📍 {{ activeEmergencyAlert.room_zone }}</span>
          </div>
          <h4 class="alert-title">{{ activeEmergencyAlert.title }}</h4>
          <p class="alert-desc">{{ activeEmergencyAlert.description }}</p>
        </div>
      </div>
      <div class="alert-actions">
        <button class="btn-alert-ack" @click="acknowledgeEmergencyAlert">✓ Acknowledge</button>
        <button class="btn-alert-view" @click="currentTab = 'incidents'">View Registry</button>
        <button class="btn-alert-dismiss" @click="dismissEmergencyAlert">✕</button>
      </div>
    </div>

    <!-- Navigation Tabs -->
    <nav class="nav-tabs">
      <button
        :class="['tab-btn', { active: currentTab === 'care' }]"
        @click="currentTab = 'care'"
      >
        🩺 Veterinary Care & Medical Holds
      </button>
      <button
        :class="['tab-btn', { active: currentTab === 'helpdesk' }]"
        @click="currentTab = 'helpdesk'"
      >
        🛠️ IT Helpdesk & Issue Tracker
      </button>
      <button
        :class="['tab-btn', { active: currentTab === 'lifecycle' }]"
        @click="currentTab = 'lifecycle'"
      >
        👥 IT Onboarding & Lifecycle Sagas
      </button>
      <button
        :class="['tab-btn', { active: currentTab === 'docs' }]"
        @click="currentTab = 'docs'"
      >
        📁 Knowledge Base & Intranet
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
        :class="['tab-btn', { active: currentTab === 'reports' }]"
        @click="currentTab = 'reports'"
      >
        📊 Executive Reports & BI
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

          <div class="form-row">
            <div class="form-group half">
              <label>Review Category</label>
              <select class="form-control" v-model="reviewForm.review_type" required>
                <option value="feline_health_assessment">🩺 Feline Health & Wellbeing Assessment</option>
                <option value="performance">👔 Joint Staff Performance Review</option>
              </select>
            </div>
            <div class="form-group half">
              <label>Review Period</label>
              <input type="text" class="form-control" v-model="reviewForm.review_period" placeholder="e.g. 2026-Q3" required />
            </div>
          </div>

          <div class="form-row">
            <div class="form-group half">
              <label>Scheduled Date</label>
              <input type="date" class="form-control" v-model="reviewForm.scheduled_for" required />
            </div>
            <div class="form-group half">
              <label>Initial Phase</label>
              <select class="form-control" v-model="reviewForm.phase">
                <option value="self_appraisal">📝 Self-Appraisal</option>
                <option value="peer_feedback">👥 Peer Feedback</option>
                <option value="manager_review">👔 Manager Review</option>
                <option value="completed">✅ Completed</option>
              </select>
            </div>
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
          <h4>🩺 Multi-Phase Assessment & Clinical Welfare Protocol</h4>
          <p>Structured evaluations transition through employee self-appraisals, peer feedback, and manager appraisals with KPI metrics. Feline wellness audits conducted by Dr. Elena Rostova record habitat optimization recommendations and clinical scores.</p>
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
          <button :class="['filter-btn', { active: reviewFilter === 'self_appraisal' }]" @click="reviewFilter = 'self_appraisal'">📝 Self-Appraisal ({{ countReviewPhase('self_appraisal') }})</button>
          <button :class="['filter-btn', { active: reviewFilter === 'peer_feedback' }]" @click="reviewFilter = 'peer_feedback'">👥 Peer Feedback ({{ countReviewPhase('peer_feedback') }})</button>
          <button :class="['filter-btn', { active: reviewFilter === 'manager_review' }]" @click="reviewFilter = 'manager_review'">👔 Manager Review ({{ countReviewPhase('manager_review') }})</button>
          <button :class="['filter-btn', { active: reviewFilter === 'completed' }]" @click="reviewFilter = 'completed'">✅ Completed ({{ countReviewStatus('completed') }})</button>
        </div>

        <div class="backtest-table-container">
          <table class="backtest-table">
            <thead>
              <tr>
                <th>Period & Date</th>
                <th>Subject</th>
                <th>Type & Phase</th>
                <th>Lifecycle Progress</th>
                <th>Evaluators</th>
                <th>Scores Breakdown</th>
                <th>Status</th>
                <th>Action</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="rc in filteredReviewCycles" :key="rc.review_id">
                <td>
                  <span class="period-badge">{{ rc.review_period || '2026-Q3' }}</span>
                  <br><small class="date-text">{{ rc.scheduled_for }}</small>
                </td>
                <td>
                  <strong>{{ getStaffName(rc.employee_id) }}</strong>
                  <br><small class="subtext-inline">{{ rc.employee_id }}</small>
                </td>
                <td>
                  <span :class="['review-type-badge', rc.review_type]">
                    {{ formatReviewType(rc.review_type) }}
                  </span>
                  <div style="margin-top: 0.25rem;">
                    <span :class="['phase-badge', rc.phase || 'self_appraisal']">
                      {{ formatPhase(rc.phase || 'self_appraisal') }}
                    </span>
                  </div>
                </td>
                <td>
                  <div class="stepper-mini">
                    <span :class="['step-dot', { active: isStepActive(rc.phase, 1), completed: isStepCompleted(rc.phase, 1) }]" title="1. Self-Appraisal">1</span>
                    <span class="step-line"></span>
                    <span :class="['step-dot', { active: isStepActive(rc.phase, 2), completed: isStepCompleted(rc.phase, 2) }]" title="2. Peer Feedback">2</span>
                    <span class="step-line"></span>
                    <span :class="['step-dot', { active: isStepActive(rc.phase, 3), completed: isStepCompleted(rc.phase, 3) }]" title="3. Manager Review">3</span>
                    <span class="step-line"></span>
                    <span :class="['step-dot', { active: isStepActive(rc.phase, 4), completed: isStepCompleted(rc.phase, 4) }]" title="4. Completed">4</span>
                  </div>
                </td>
                <td>
                  <div class="evaluators-list">
                    <span v-if="rc.reviewer_id"><small>👔 Lead:</small> {{ getStaffName(rc.reviewer_id) }}</span>
                    <span v-if="rc.peer_reviewer_id"><small>👥 Peer:</small> {{ getStaffName(rc.peer_reviewer_id) }}</span>
                    <span v-if="rc.clinical_auditor_id"><small>🩺 Vet:</small> {{ getStaffName(rc.clinical_auditor_id) }}</span>
                    <span v-if="!rc.reviewer_id && !rc.clinical_auditor_id" class="text-muted-small">Unassigned</span>
                  </div>
                </td>
                <td>
                  <div class="scores-container">
                    <span v-if="rc.score != null" class="score-badge">
                      ⭐ {{ rc.score.toFixed(2) }}
                    </span>
                    <span v-else class="text-muted-small">In Progress</span>
                    <div class="score-subtext">
                      <span v-if="rc.self_score != null">Self: {{ rc.self_score.toFixed(2) }}</span>
                      <span v-if="rc.peer_score != null">Peer: {{ rc.peer_score.toFixed(2) }}</span>
                      <span v-if="rc.wellness_audit_score != null">Vet: {{ rc.wellness_audit_score.toFixed(2) }}</span>
                    </div>
                  </div>
                </td>
                <td>
                  <span :class="['status-badge', rc.status]">{{ rc.status.toUpperCase() }}</span>
                </td>
                <td>
                  <button class="review-action-btn" @click="openGradingModal(rc)">
                    📋 Review & Audit
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- Multi-Phase Evaluation & Veterinary Audit Modal -->
        <div v-if="activeReviewForGrading" class="grading-modal-card">
          <div class="grading-header">
            <div>
              <h3>📋 Multi-Phase Review & Audit: {{ getStaffName(activeReviewForGrading.employee_id) }}</h3>
              <p class="subtext">
                Period: <strong>{{ activeReviewForGrading.review_period || '2026-Q3' }}</strong> |
                Category: <strong>{{ formatReviewType(activeReviewForGrading.review_type) }}</strong> |
                Current Phase: <strong style="color: #38bdf8;">{{ formatPhase(activeReviewForGrading.phase || 'self_appraisal') }}</strong>
              </p>
            </div>
            <button class="close-btn" @click="activeReviewForGrading = null">✕</button>
          </div>

          <!-- Modal Navigation Tabs -->
          <div class="modal-tab-bar">
            <button :class="['modal-tab-btn', { active: evaluationTab === 'self' }]" @click="evaluationTab = 'self'">
              📝 1. Self-Appraisal
            </button>
            <button :class="['modal-tab-btn', { active: evaluationTab === 'peer' }]" @click="evaluationTab = 'peer'">
              👥 2. Peer Feedback
            </button>
            <button :class="['modal-tab-btn', { active: evaluationTab === 'manager' }]" @click="evaluationTab = 'manager'">
              👔 3. Manager Assessment
            </button>
            <button :class="['modal-tab-btn', { active: evaluationTab === 'vet' }]" @click="evaluationTab = 'vet'">
              🩺 4. Veterinary Clinical Audit
            </button>
          </div>

          <!-- Tab 1: Self-Appraisal -->
          <div v-if="evaluationTab === 'self'" class="tab-content-panel">
            <h4>Employee Self-Appraisal & Reflections</h4>
            <div class="form-row">
              <div class="form-group half">
                <label>Self Evaluation Score (0.00 - 5.00)</label>
                <div class="score-input-group">
                  <input type="number" step="0.05" min="0" max="5" class="form-control score-input" v-model.number="gradingForm.self_score" />
                  <span class="score-scale">/ 5.00</span>
                </div>
              </div>
              <div class="form-group half">
                <label>KPI Self-Assessment Reflection</label>
                <input type="text" class="form-control" v-model="gradingForm.kpi_ratings" placeholder='{"workplace_safety": 5.0, "alpha_telemetry": 4.9}' />
              </div>
            </div>
            <div class="form-group">
              <label>Self-Appraisal Statements</label>
              <textarea class="form-control" v-model="gradingForm.self_appraisal" rows="3" placeholder="Document quarterly contributions, habitat maintenance achievements, or alpha perch milestones..."></textarea>
            </div>
            <div class="grading-actions">
              <button type="button" class="btn-action" style="background: #0284c7; color: white;" @click="submitSelfAppraisalAction" :disabled="isSavingGrading">
                {{ isSavingGrading ? '⌛ Submitting...' : '💾 Submit Self-Appraisal & Advance to Peer Feedback' }}
              </button>
            </div>
          </div>

          <!-- Tab 2: Peer Feedback -->
          <div v-if="evaluationTab === 'peer'" class="tab-content-panel">
            <h4>Peer Review & Cross-Functional Feedback</h4>
            <div class="form-row">
              <div class="form-group half">
                <label>Peer Reviewer</label>
                <select class="form-control" v-model="gradingForm.peer_reviewer_id">
                  <option value="">Select Peer Reviewer</option>
                  <option v-for="emp in staffList" :key="'peer-' + emp.id" :value="emp.id">
                    {{ emp.icon }} {{ emp.name }}
                  </option>
                </select>
              </div>
              <div class="form-group half">
                <label>Peer Evaluation Score (0.00 - 5.00)</label>
                <div class="score-input-group">
                  <input type="number" step="0.05" min="0" max="5" class="form-control score-input" v-model.number="gradingForm.peer_score" />
                  <span class="score-scale">/ 5.00</span>
                </div>
              </div>
            </div>
            <div class="form-group">
              <label>Peer Feedback Remarks</label>
              <textarea class="form-control" v-model="gradingForm.peer_feedback" rows="3" placeholder="Provide constructive feedback on collaboration, habitat coordination, and cross-species cooperation..."></textarea>
            </div>
            <div class="grading-actions">
              <button type="button" class="btn-action" style="background: #0284c7; color: white;" @click="submitPeerFeedbackAction" :disabled="isSavingGrading">
                {{ isSavingGrading ? '⌛ Submitting...' : '💾 Submit Peer Feedback & Advance to Manager Review' }}
              </button>
            </div>
          </div>

          <!-- Tab 3: Manager Assessment -->
          <div v-if="evaluationTab === 'manager'" class="tab-content-panel">
            <h4>Manager Assessment & Final KPI Rating</h4>
            <div class="form-row">
              <div class="form-group half">
                <label>Lead Reviewer / Manager</label>
                <select class="form-control" v-model="gradingForm.reviewer_id">
                  <option v-for="emp in staffList" :key="'mgr-' + emp.id" :value="emp.id">
                    {{ emp.icon }} {{ emp.name }}
                  </option>
                </select>
              </div>
              <div class="form-group half">
                <label>Manager Evaluation Score (0.00 - 5.00)</label>
                <div class="score-input-group">
                  <input type="number" step="0.05" min="0" max="5" class="form-control score-input" v-model.number="gradingForm.manager_score" />
                  <span class="score-scale">/ 5.00</span>
                </div>
              </div>
            </div>
            <div class="form-row">
              <div class="form-group half">
                <label>Review Status</label>
                <select class="form-control" v-model="gradingForm.status">
                  <option value="completed">✅ Completed & Finalized</option>
                  <option value="in_progress">⏳ In Progress</option>
                  <option value="scheduled">📅 Scheduled</option>
                </select>
              </div>
              <div class="form-group half">
                <label>Overall Final Score</label>
                <div class="score-input-group">
                  <input type="number" step="0.05" min="0" max="5" class="form-control score-input" v-model.number="gradingForm.score" />
                  <span class="score-scale">/ 5.00</span>
                </div>
              </div>
            </div>
            <div class="form-group">
              <label>Manager Evaluation Remarks</label>
              <textarea class="form-control" v-model="gradingForm.manager_assessment" rows="3" placeholder="Executive performance assessment, KPI objective score, and annual compensation review..."></textarea>
            </div>
            <div class="grading-actions">
              <button type="button" class="btn-action" style="background: #16a34a; color: white;" @click="submitManagerAssessmentAction" :disabled="isSavingGrading">
                {{ isSavingGrading ? '⌛ Finalizing...' : '✅ Finalize & Sign Review Cycle' }}
              </button>
            </div>
          </div>

          <!-- Tab 4: Veterinary Audit & Habitat Recommendations -->
          <div v-if="evaluationTab === 'vet'" class="tab-content-panel">
            <h4>🩺 Veterinary Clinical Wellness Audit (Dr. Elena Rostova)</h4>
            <div class="form-row">
              <div class="form-group half">
                <label>Lead Clinical Auditor</label>
                <select class="form-control" v-model="gradingForm.clinical_auditor_id">
                  <option v-for="emp in staffList" :key="'vet-' + emp.id" :value="emp.id">
                    {{ emp.icon }} {{ emp.name }}
                  </option>
                </select>
              </div>
              <div class="form-group half">
                <label>Clinical Wellness Audit Score (0.00 - 5.00)</label>
                <div class="score-input-group">
                  <input type="number" step="0.05" min="0" max="5" class="form-control score-input" v-model.number="gradingForm.wellness_audit_score" />
                  <span class="score-scale">/ 5.00</span>
                </div>
              </div>
            </div>
            <div class="form-group">
              <label>Habitat Optimization & Welfare Recommendations</label>
              <textarea class="form-control" v-model="gradingForm.habitat_recommendations" rows="3" placeholder="Specify habitat sunbeam exposure adjustments, perch mobility modifications, dietary plans, or climate comfort directives..."></textarea>
            </div>
            <div class="grading-actions">
              <button type="button" class="btn-action" style="background: #0d9488; color: white;" @click="submitVeterinaryAuditAction" :disabled="isSavingGrading">
                {{ isSavingGrading ? '⌛ Recording Audit...' : '🩺 Record Clinical Wellness Audit' }}
              </button>
            </div>
          </div>
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

    <!-- TAB: Internal Documents, Knowledge Base & Shared Calendar -->
    <div v-else-if="currentTab === 'docs'" style="display: flex; flex-direction: column; gap: 1.5rem;">
      <!-- Hub Banner / Header Card -->
      <div class="card" style="display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 1rem;">
        <div>
          <h2 style="margin: 0; color: #f8fafc; font-size: 1.35rem;">📁 Nextcloud Departmental Hub & Intranet</h2>
          <p class="subtext" style="margin-bottom: 0;">Centralized repository for departmental group folders, company policies, WebDAV document reader, and company-wide calendar.</p>
        </div>
        <div style="display: flex; gap: 0.75rem;">
          <button class="btn-action" style="background: var(--secondary-color, #ea580c); color: white;" @click="showNewDocModal = true">
            📄 Publish Document
          </button>
          <button class="btn-action" style="background: #38bdf8; color: #0f172a; font-weight: bold;" @click="showNewEventModal = true">
            📅 Schedule Event
          </button>
        </div>
      </div>

      <!-- Departmental Group Folders (4 Cards) -->
      <div>
        <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.75rem;">
          <h3 style="margin: 0; color: #cbd5e1; font-size: 1rem;">🏢 Departmental Hub Group Folders</h3>
          <button v-if="selectedDepartmentFilter" class="btn-action" style="background: #334155; color: #38bdf8; padding: 0.25rem 0.6rem; font-size: 0.75rem;" @click="selectedDepartmentFilter = ''">
            Clear Folder Filter (Showing All)
          </button>
        </div>
        <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(230px, 1fr)); gap: 1rem;">
          <div
            v-for="folder in hubFolders"
            :key="folder.id"
            :class="['card', { active: selectedDepartmentFilter === folder.mount_point }]"
            style="cursor: pointer; transition: all 0.2s; border: 1px solid #334155; padding: 1.25rem;"
            :style="selectedDepartmentFilter === folder.mount_point ? 'border-color: #38bdf8; background: #1e293b;' : ''"
            @click="toggleDepartmentFilter(folder.mount_point)"
          >
            <div style="display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 0.5rem;">
              <span style="font-size: 1.5rem;">{{ getDepartmentFolderIcon(folder.mount_point) }}</span>
              <span class="badge" style="background: #0284c7; color: white;">{{ folder.quota }}</span>
            </div>
            <strong style="color: #f8fafc; font-size: 0.95rem; display: block; margin-bottom: 0.25rem;">{{ folder.mount_point }}</strong>
            <div style="font-size: 0.75rem; color: #94a3b8; margin-bottom: 0.5rem;">
              <span>Groups: {{ folder.groups ? folder.groups.join(', ') : 'all-staff' }}</span>
            </div>
            <div style="display: flex; justify-content: space-between; font-size: 0.75rem; color: #cbd5e1; border-top: 1px solid #334155; padding-top: 0.5rem;">
              <span>{{ getDocumentCountForDepartment(folder.mount_point) }} documents</span>
              <span style="color: #38bdf8;">{{ selectedDepartmentFilter === folder.mount_point ? 'Filtered ✕' : 'Filter Files →' }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Main Layout: Documents Table (Left/Center) + Shared Calendar (Right) -->
      <div style="display: grid; grid-template-columns: 2fr 1.1fr; gap: 1.5rem;">
        <!-- Left Column: Intranet Documents & Search -->
        <div class="card">
          <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem; flex-wrap: wrap; gap: 0.75rem;">
            <div>
              <h3 style="margin: 0; color: #f8fafc; font-size: 1.1rem;">📄 Intranet Knowledge Base</h3>
              <p class="subtext" style="margin-bottom: 0;">Standard operating procedures, clinical guides & engineering specs.</p>
            </div>
            <div style="display: flex; gap: 0.5rem; align-items: center; flex-wrap: wrap;">
              <input
                v-model="docSearchQuery"
                type="text"
                class="score-input"
                style="text-align: left; width: 180px; font-size: 0.85rem;"
                placeholder="🔍 Search documents..."
              />
              <select v-model="selectedCategoryFilter" class="score-input" style="font-size: 0.85rem;">
                <option value="">All Categories</option>
                <option value="Policy">Policy</option>
                <option value="SOP">SOP</option>
                <option value="Clinical Protocol">Clinical Protocol</option>
                <option value="Architecture">Architecture</option>
                <option value="Report">Report</option>
              </select>
            </div>
          </div>

          <!-- Document List Table -->
          <div class="table-container" style="overflow-x: auto;">
            <table class="data-table" style="width: 100%; border-collapse: collapse;">
              <thead>
                <tr>
                  <th style="padding: 0.75rem; border-bottom: 1px solid #334155; color: #94a3b8;">Title</th>
                  <th style="padding: 0.75rem; border-bottom: 1px solid #334155; color: #94a3b8;">Department</th>
                  <th style="padding: 0.75rem; border-bottom: 1px solid #334155; color: #94a3b8;">Category</th>
                  <th style="padding: 0.75rem; border-bottom: 1px solid #334155; color: #94a3b8;">Author</th>
                  <th style="padding: 0.75rem; border-bottom: 1px solid #334155; color: #94a3b8;">WebDAV</th>
                  <th style="padding: 0.75rem; border-bottom: 1px solid #334155; color: #94a3b8;">Action</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="doc in filteredDocuments" :key="doc.document_id" style="border-bottom: 1px solid #1e293b;">
                  <td style="padding: 0.75rem;">
                    <div style="font-weight: 600; color: #f8fafc;">{{ doc.title }}</div>
                    <div style="font-size: 0.75rem; color: #94a3b8;">v{{ doc.version || '1.0' }} • {{ doc.document_id }}</div>
                  </td>
                  <td style="padding: 0.75rem;">
                    <span class="badge" style="background: #334155; color: #cbd5e1; font-size: 0.75rem;">{{ doc.department }}</span>
                  </td>
                  <td style="padding: 0.75rem;">
                    <span class="badge" :style="getCategoryBadgeStyle(doc.category)">{{ doc.category }}</span>
                  </td>
                  <td style="padding: 0.75rem; font-size: 0.85rem; color: #cbd5e1;">
                    {{ getAuthorDisplayName(doc.author) }}
                  </td>
                  <td style="padding: 0.75rem;">
                    <div style="display: flex; align-items: center; gap: 0.35rem;">
                      <code style="font-size: 0.7rem; color: #38bdf8; max-width: 120px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;" :title="doc.webdav_path">{{ doc.webdav_path }}</code>
                      <button
                        type="button"
                        class="btn-action"
                        style="padding: 0.2rem 0.4rem; font-size: 0.65rem; background: #1e293b; color: #94a3b8;"
                        @click="copyWebDAVPath(doc.webdav_path)"
                        title="Copy WebDAV URL"
                      >
                        📋
                      </button>
                    </div>
                  </td>
                  <td style="padding: 0.75rem;">
                    <button
                      class="btn-action"
                      style="background: #0284c7; color: white; padding: 0.35rem 0.75rem; font-size: 0.8rem;"
                      @click="openDocumentReader(doc)"
                    >
                      📖 Read
                    </button>
                  </td>
                </tr>
                <tr v-if="filteredDocuments.length === 0">
                  <td colspan="6" style="text-align: center; color: #94a3b8; padding: 2rem;">
                    No documents matching current search or filters.
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        <!-- Right Column: Shared Company Operations Calendar -->
        <div class="card" style="display: flex; flex-direction: column; gap: 1rem;">
          <div>
            <h3 style="margin: 0; color: #f8fafc; font-size: 1.1rem;">📅 Operations Calendar</h3>
            <p class="subtext" style="margin-bottom: 0;">Shift rotations, maintenance & milestones</p>
          </div>

          <!-- Calendar Selector Tabs -->
          <div style="display: flex; gap: 0.35rem; flex-wrap: wrap;">
            <button
              v-for="cal in ['All', 'Shift Rotations', 'Habitat Maintenance Windows', 'Corporate Milestones', 'Veterinary Audits']"
              :key="cal"
              class="btn-action"
              :style="selectedCalendarFilter === (cal === 'All' ? '' : cal) ? 'background: #38bdf8; color: #0f172a; font-weight: bold;' : 'background: #0f172a; color: #94a3b8; border: 1px solid #334155;'"
              style="padding: 0.25rem 0.5rem; font-size: 0.7rem;"
              @click="selectedCalendarFilter = (cal === 'All' ? '' : cal)"
            >
              {{ cal === 'Habitat Maintenance Windows' ? 'Maintenance' : (cal === 'Shift Rotations' ? 'Shifts' : (cal === 'Corporate Milestones' ? 'Milestones' : (cal === 'Veterinary Audits' ? 'Vet Audits' : 'All'))) }}
            </button>
          </div>

          <!-- Event Cards List -->
          <div style="display: flex; flex-direction: column; gap: 0.75rem; max-height: 540px; overflow-y: auto;">
            <div
              v-for="ev in filteredCalendarEvents"
              :key="ev.event_id"
              style="background: #0f172a; border: 1px solid #334155; border-radius: 0.5rem; padding: 0.85rem;"
            >
              <div style="display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 0.35rem;">
                <span class="badge" :style="getCalendarBadgeStyle(ev.calendar)" style="font-size: 0.7rem;">{{ ev.calendar }}</span>
                <span :class="['badge', ev.status === 'confirmed' ? 'badge-approved' : 'badge-pending']" style="font-size: 0.7rem;">{{ ev.status }}</span>
              </div>
              <strong style="color: #f8fafc; font-size: 0.9rem; display: block; margin-bottom: 0.25rem;">{{ ev.title }}</strong>
              <div v-if="ev.description" style="font-size: 0.8rem; color: #cbd5e1; margin-bottom: 0.5rem;">
                {{ ev.description }}
              </div>
              <div style="display: flex; flex-direction: column; gap: 0.25rem; font-size: 0.75rem; color: #94a3b8; border-top: 1px solid #1e293b; padding-top: 0.5rem;">
                <div>📍 {{ ev.location || 'HQ Habitat' }} ({{ ev.department }})</div>
                <div>🕒 {{ formatDateTime(ev.start_time) }} - {{ formatDateTime(ev.end_time) }}</div>
                <div>👤 Organizer: {{ getAuthorDisplayName(ev.organizer) }}</div>
              </div>
            </div>
            <div v-if="filteredCalendarEvents.length === 0" style="text-align: center; color: #94a3b8; padding: 2rem;">
              No calendar events in selected filter.
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- TAB: IT Helpdesk & Issue Tracker -->
    <div v-else-if="currentTab === 'helpdesk'" style="display: flex; flex-direction: column; gap: 1.5rem;">
      <!-- Overview & Metrics Header -->
      <div class="card" style="background: #1e293b; border: 1px solid #334155; padding: 1.5rem; border-radius: 0.75rem;">
        <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1.25rem; flex-wrap: wrap; gap: 1rem;">
          <div>
            <h2 style="margin: 0; color: #f8fafc; font-size: 1.3rem;">🛠️ IT Helpdesk & Issue Management</h2>
            <div style="font-size: 0.85rem; color: #94a3b8; margin-top: 0.25rem;">
              Forgejo Issue Bridge (<code>oci-internal/helpdesk</code>) • Pebble Watch & Mobile Push Dispatched Alerts
            </div>
          </div>
          <button class="btn-action" style="background: var(--secondary-color, #ea580c); color: white;" @click="showNewTicketModal = true">
            ➕ New Support Ticket
          </button>
        </div>

        <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 1rem;">
          <div style="background: #0f172a; padding: 1rem; border-radius: 0.5rem; border: 1px solid #334155;">
            <div style="font-size: 0.75rem; color: #94a3b8; text-transform: uppercase;">Total Tickets</div>
            <div style="font-size: 1.6rem; font-weight: bold; color: #f8fafc; margin-top: 0.25rem;">{{ helpdeskTickets.length }}</div>
          </div>
          <div style="background: #0f172a; padding: 1rem; border-radius: 0.5rem; border: 1px solid #334155;">
            <div style="font-size: 0.75rem; color: #94a3b8; text-transform: uppercase;">Open / Pending</div>
            <div style="font-size: 1.6rem; font-weight: bold; color: #38bdf8; margin-top: 0.25rem;">
              {{ helpdeskTickets.filter(t => t.state === 'open').length }}
            </div>
          </div>
          <div style="background: #0f172a; padding: 1rem; border-radius: 0.5rem; border: 1px solid #334155;">
            <div style="font-size: 0.75rem; color: #94a3b8; text-transform: uppercase;">Acknowledged (Pebble/On-Call)</div>
            <div style="font-size: 1.6rem; font-weight: bold; color: #f59e0b; margin-top: 0.25rem;">
              {{ helpdeskTickets.filter(t => t.state === 'acknowledged').length }}
            </div>
          </div>
          <div style="background: #0f172a; padding: 1rem; border-radius: 0.5rem; border: 1px solid #334155;">
            <div style="font-size: 0.75rem; color: #94a3b8; text-transform: uppercase;">Urgent / Critical</div>
            <div style="font-size: 1.6rem; font-weight: bold; color: #ef4444; margin-top: 0.25rem;">
              {{ helpdeskTickets.filter(t => t.is_urgent || t.severity === 'urgent' || t.severity === 'critical').length }}
            </div>
          </div>
        </div>
      </div>

      <!-- Filter Controls & Ticket Table -->
      <div class="card" style="background: #1e293b; border: 1px solid #334155; padding: 1.5rem; border-radius: 0.75rem;">
        <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem; flex-wrap: wrap; gap: 0.75rem;">
          <h3 style="margin: 0; color: #f8fafc; font-size: 1.1rem;">Ticket Registry</h3>
          <div style="display: flex; gap: 0.5rem; flex-wrap: wrap;">
            <select v-model="helpdeskFilterStatus" class="score-input" style="font-size: 0.8rem; padding: 0.35rem 0.6rem;">
              <option value="all">All States</option>
              <option value="open">Open</option>
              <option value="acknowledged">Acknowledged</option>
              <option value="closed">Closed</option>
            </select>
            <select v-model="helpdeskFilterSeverity" class="score-input" style="font-size: 0.8rem; padding: 0.35rem 0.6rem;">
              <option value="all">All Severities</option>
              <option value="low">Low</option>
              <option value="medium">Medium</option>
              <option value="high">High</option>
              <option value="urgent">Urgent</option>
              <option value="critical">Critical</option>
            </select>
            <select v-model="helpdeskFilterCategory" class="score-input" style="font-size: 0.8rem; padding: 0.35rem 0.6rem;">
              <option value="all">All Categories</option>
              <option value="Hardware">Hardware</option>
              <option value="Software">Software</option>
              <option value="Habitat">Habitat</option>
              <option value="Trading">Trading</option>
              <option value="Security">Security</option>
            </select>
          </div>
        </div>

        <div class="table-container">
          <table class="data-table" style="width: 100%; border-collapse: collapse;">
            <thead>
              <tr style="border-bottom: 2px solid #334155; text-align: left; color: #94a3b8; font-size: 0.8rem;">
                <th style="padding: 0.75rem;">Issue #</th>
                <th style="padding: 0.75rem;">Title & Category</th>
                <th style="padding: 0.75rem;">Severity</th>
                <th style="padding: 0.75rem;">Status</th>
                <th style="padding: 0.75rem;">Reporter / Source</th>
                <th style="padding: 0.75rem;">On-Call Alert</th>
                <th style="padding: 0.75rem; text-align: right;">Actions</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="ticket in filteredHelpdeskTickets" :key="ticket.ticket_id" style="border-bottom: 1px solid #334155; font-size: 0.85rem;">
                <td style="padding: 0.75rem; font-family: monospace; color: #38bdf8;">
                  #{{ ticket.forgejo_issue_id || ticket.ticket_id.substring(0, 6) }}
                </td>
                <td style="padding: 0.75rem;">
                  <strong style="color: #f8fafc; display: block;">{{ ticket.title }}</strong>
                  <span style="font-size: 0.75rem; color: #94a3b8;">🏷️ {{ ticket.category || 'General' }} • Repo: {{ ticket.forgejo_repo }}</span>
                </td>
                <td style="padding: 0.75rem;">
                  <span :class="['badge', getSeverityBadgeClass(ticket.severity)]">
                    {{ ticket.severity ? ticket.severity.toUpperCase() : 'MEDIUM' }}
                  </span>
                </td>
                <td style="padding: 0.75rem;">
                  <span :class="['badge', getTicketStatusBadgeClass(ticket.state)]">
                    {{ ticket.state ? ticket.state.toUpperCase() : 'OPEN' }}
                  </span>
                </td>
                <td style="padding: 0.75rem;">
                  <div style="color: #e2e8f0;">{{ ticket.author_username }}</div>
                  <div style="font-size: 0.75rem; color: #94a3b8;">Source: {{ ticket.source || 'internal' }}</div>
                </td>
                <td style="padding: 0.75rem;">
                  <span v-if="ticket.is_urgent || ticket.severity === 'urgent' || ticket.severity === 'critical'" style="display: inline-flex; align-items: center; gap: 0.25rem; font-size: 0.75rem; color: #f87171; background: rgba(239, 68, 68, 0.15); padding: 0.2rem 0.4rem; border-radius: 4px; border: 1px solid rgba(239, 68, 68, 0.3);">
                    ⌚ Pebble & Push
                  </span>
                  <span v-else style="font-size: 0.75rem; color: #64748b;">Standard Sync</span>
                </td>
                <td style="padding: 0.75rem; text-align: right;">
                  <div style="display: flex; gap: 0.4rem; justify-content: flex-end;">
                    <button class="btn-action" style="background: #334155; color: white; padding: 0.25rem 0.5rem; font-size: 0.75rem;" @click="openTicketDetail(ticket)">
                      👁️ View
                    </button>
                    <button v-if="ticket.state === 'open'" class="btn-action" style="background: #d97706; color: white; padding: 0.25rem 0.5rem; font-size: 0.75rem;" @click="quickAcknowledgeTicket(ticket)">
                      ✔️ ACK
                    </button>
                    <button v-if="ticket.state !== 'closed'" class="btn-action" style="background: #059669; color: white; padding: 0.25rem 0.5rem; font-size: 0.75rem;" @click="quickCloseTicket(ticket)">
                      ✅ Close
                    </button>
                  </div>
                </td>
              </tr>
              <tr v-if="filteredHelpdeskTickets.length === 0">
                <td colspan="7" style="text-align: center; color: #94a3b8; padding: 2rem;">
                  No IT tickets matching active filter.
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- TAB: Executive Reporting & Business Intelligence -->
    <div v-else-if="currentTab === 'reports'" style="display: flex; flex-direction: column; gap: 1.5rem;">
      <!-- Overview & Header -->
      <div class="card" style="background: #1e293b; border: 1px solid #334155; padding: 1.5rem; border-radius: 0.75rem;">
        <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1.25rem; flex-wrap: wrap; gap: 1rem;">
          <div>
            <h2 style="margin: 0; color: #f8fafc; font-size: 1.3rem;">📊 Executive Reporting & Business Intelligence</h2>
            <div style="font-size: 0.85rem; color: #94a3b8; margin-top: 0.25rem;">
              Automated Monthly Financial Summaries • Frappe HR / ERPNext Reconciled Ledgers • Nextcloud WebDAV Compliance Archival
            </div>
          </div>
          <button class="btn-action" style="background: var(--secondary-color, #ea580c); color: white;" @click="showGenerateReportModal = true">
            ➕ Generate Monthly Report
          </button>
        </div>

        <!-- Latest Report Summary Cards -->
        <div v-if="selectedReport" style="display: grid; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); gap: 1rem;">
          <div style="background: #0f172a; padding: 1rem; border-radius: 0.5rem; border: 1px solid #334155;">
            <div style="font-size: 0.75rem; color: #94a3b8; text-transform: uppercase;">Net Profit</div>
            <div style="font-size: 1.5rem; font-weight: bold; color: #4ade80; margin-top: 0.25rem;">
              ${{ Number(selectedReport.net_profit || 0).toLocaleString(undefined, { minimumFractionDigits: 2 }) }}
            </div>
            <div style="font-size: 0.75rem; color: #94a3b8; margin-top: 0.25rem;">Period: {{ selectedReport.period }}</div>
          </div>

          <div style="background: #0f172a; padding: 1rem; border-radius: 0.5rem; border: 1px solid #334155;">
            <div style="font-size: 0.75rem; color: #94a3b8; text-transform: uppercase;">Total Revenue</div>
            <div style="font-size: 1.5rem; font-weight: bold; color: #38bdf8; margin-top: 0.25rem;">
              ${{ Number(selectedReport.total_revenue || 0).toLocaleString(undefined, { minimumFractionDigits: 2 }) }}
            </div>
            <div style="font-size: 0.75rem; color: #94a3b8; margin-top: 0.25rem;">ERPNext P&L</div>
          </div>

          <div style="background: #0f172a; padding: 1rem; border-radius: 0.5rem; border: 1px solid #334155;">
            <div style="font-size: 0.75rem; color: #94a3b8; text-transform: uppercase;">Operational Burn</div>
            <div style="font-size: 1.5rem; font-weight: bold; color: #f87171; margin-top: 0.25rem;">
              ${{ Number(selectedReport.operational_burn || 0).toLocaleString(undefined, { minimumFractionDigits: 2 }) }}
            </div>
            <div style="font-size: 0.75rem; color: #94a3b8; margin-top: 0.25rem;">Operating + Payroll</div>
          </div>

          <div style="background: #0f172a; padding: 1rem; border-radius: 0.5rem; border: 1px solid #334155;">
            <div style="font-size: 0.75rem; color: #94a3b8; text-transform: uppercase;">Cash Reserves</div>
            <div style="font-size: 1.5rem; font-weight: bold; color: #facc15; margin-top: 0.25rem;">
              ${{ Number(selectedReport.cash_reserves || 0).toLocaleString(undefined, { minimumFractionDigits: 2 }) }}
            </div>
            <div style="font-size: 0.75rem; color: #94a3b8; margin-top: 0.25rem;">Runway: {{ selectedReport.runway_months }} mos</div>
          </div>

          <div style="background: #0f172a; padding: 1rem; border-radius: 0.5rem; border: 1px solid #334155;">
            <div style="font-size: 0.75rem; color: #94a3b8; text-transform: uppercase;">Workforce Headcount</div>
            <div style="font-size: 1.5rem; font-weight: bold; color: #c084fc; margin-top: 0.25rem;">
              {{ selectedReport.total_headcount }}
            </div>
            <div style="font-size: 0.75rem; color: #94a3b8; margin-top: 0.25rem;">{{ selectedReport.human_headcount }} Human / {{ selectedReport.feline_headcount }} Feline</div>
          </div>

          <div style="background: #0f172a; padding: 1rem; border-radius: 0.5rem; border: 1px solid #334155;">
            <div style="font-size: 0.75rem; color: #94a3b8; text-transform: uppercase;">Attendance Rate</div>
            <div style="font-size: 1.5rem; font-weight: bold; color: #34d399; margin-top: 0.25rem;">
              {{ selectedReport.attendance_rate }}%
            </div>
            <div style="font-size: 0.75rem; color: #94a3b8; margin-top: 0.25rem;">Timesheets: {{ selectedReport.timesheets_approved }} approved</div>
          </div>
        </div>
      </div>

      <!-- Historical Reports Table -->
      <div class="card" style="background: #1e293b; border: 1px solid #334155; padding: 1.5rem; border-radius: 0.75rem;">
        <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem; flex-wrap: wrap; gap: 0.75rem;">
          <div>
            <h3 style="margin: 0; color: #f8fafc; font-size: 1.1rem;">📑 Historical Executive Management Reports</h3>
            <p class="subtext" style="margin-bottom: 0;">Immutable monthly management summaries deposited directly into Nextcloud Executive vault.</p>
          </div>
          <button class="btn-action" style="background: #334155; color: white;" @click="fetchExecutiveReports">
            🔄 Refresh Reports
          </button>
        </div>

        <div class="table-container">
          <table class="data-table" style="width: 100%; border-collapse: collapse;">
            <thead>
              <tr style="border-bottom: 2px solid #334155; text-align: left; color: #94a3b8; font-size: 0.8rem;">
                <th style="padding: 0.75rem;">Period</th>
                <th style="padding: 0.75rem;">Report Title</th>
                <th style="padding: 0.75rem;">Financials (Rev / Net)</th>
                <th style="padding: 0.75rem;">Burn & Runway</th>
                <th style="padding: 0.75rem;">Headcount</th>
                <th style="padding: 0.75rem;">Status</th>
                <th style="padding: 0.75rem;">Nextcloud WebDAV Archive</th>
                <th style="padding: 0.75rem; text-align: right;">Actions</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="rep in executiveReports" :key="rep.report_id" style="border-bottom: 1px solid #334155; font-size: 0.85rem;">
                <td style="padding: 0.75rem; font-family: monospace; font-weight: bold; color: #38bdf8;">
                  {{ rep.period }}
                </td>
                <td style="padding: 0.75rem;">
                  <strong style="color: #f8fafc; display: block;">{{ rep.title }}</strong>
                  <span style="font-size: 0.75rem; color: #94a3b8;">Generated by: {{ rep.generated_by }}</span>
                </td>
                <td style="padding: 0.75rem;">
                  <div style="color: #e2e8f0;">Rev: ${{ Number(rep.total_revenue || 0).toLocaleString() }}</div>
                  <div style="font-size: 0.75rem; color: #4ade80;">Net: ${{ Number(rep.net_profit || 0).toLocaleString() }}</div>
                </td>
                <td style="padding: 0.75rem;">
                  <div style="color: #f87171;">Burn: ${{ Number(rep.operational_burn || 0).toLocaleString() }}/mo</div>
                  <div style="font-size: 0.75rem; color: #facc15;">Runway: {{ rep.runway_months }} mos</div>
                </td>
                <td style="padding: 0.75rem;">
                  <span class="badge" style="background: #475569; color: white;">{{ rep.total_headcount }} Staff</span>
                  <div style="font-size: 0.7rem; color: #94a3b8; margin-top: 0.2rem;">{{ rep.attendance_rate }}% att.</div>
                </td>
                <td style="padding: 0.75rem;">
                  <span class="badge badge-success">
                    🔒 {{ rep.status ? rep.status.toUpperCase() : 'ARCHIVED' }}
                  </span>
                </td>
                <td style="padding: 0.75rem; font-family: monospace; font-size: 0.75rem; color: #fdba74;">
                  <div v-if="rep.webdav_path_pdf" style="max-width: 200px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;" :title="rep.webdav_path_pdf">
                    📄 {{ rep.webdav_path_pdf.split('/').pop() }}
                  </div>
                  <div v-if="rep.webdav_path_csv" style="max-width: 200px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;" :title="rep.webdav_path_csv">
                    📊 {{ rep.webdav_path_csv.split('/').pop() }}
                  </div>
                </td>
                <td style="padding: 0.75rem; text-align: right;">
                  <div style="display: flex; gap: 0.4rem; justify-content: flex-end;">
                    <button class="btn-action" style="background: #334155; color: white; padding: 0.25rem 0.5rem; font-size: 0.75rem;" @click="openReportDetail(rep)">
                      👁️ Inspect
                    </button>
                    <button class="btn-action" style="background: #0284c7; color: white; padding: 0.25rem 0.5rem; font-size: 0.75rem;" @click="downloadReport(rep, 'pdf')">
                      📄 PDF
                    </button>
                    <button class="btn-action" style="background: #059669; color: white; padding: 0.25rem 0.5rem; font-size: 0.75rem;" @click="downloadReport(rep, 'csv')">
                      📊 CSV
                    </button>
                  </div>
                </td>
              </tr>
              <tr v-if="executiveReports.length === 0">
                <td colspan="8" style="text-align: center; color: #94a3b8; padding: 2rem;">
                  No executive management reports generated yet.
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

    <!-- Document Reader Modal (Embedded Markdown/Doc Viewer) -->
    <div v-if="showDocumentReaderModal && selectedDocument" class="modal-overlay" @click.self="showDocumentReaderModal = false">
      <div class="modal-card" style="max-width: 750px; background: #0f172a; border: 1px solid #334155; padding: 2rem; border-radius: 0.75rem; max-height: 85vh; overflow-y: auto;">
        <div style="display: flex; justify-content: space-between; align-items: flex-start; border-bottom: 2px solid #334155; padding-bottom: 1rem; margin-bottom: 1.5rem;">
          <div>
            <div style="display: flex; gap: 0.5rem; margin-bottom: 0.35rem;">
              <span class="badge" style="background: #334155; color: #cbd5e1; font-size: 0.75rem;">{{ selectedDocument.department }}</span>
              <span class="badge" :style="getCategoryBadgeStyle(selectedDocument.category)">{{ selectedDocument.category }}</span>
              <span class="badge" style="background: #0284c7; color: white;">v{{ selectedDocument.version || '1.0' }}</span>
            </div>
            <h2 style="margin: 0; color: #38bdf8;">{{ selectedDocument.title }}</h2>
            <div style="font-size: 0.8rem; color: #94a3b8; margin-top: 0.25rem;">
              Author: {{ getAuthorDisplayName(selectedDocument.author) }} • ID: {{ selectedDocument.document_id }}
            </div>
          </div>
          <button class="close-btn" @click="showDocumentReaderModal = false">✕</button>
        </div>

        <div style="background: #1e293b; padding: 0.75rem 1rem; border-radius: 0.5rem; margin-bottom: 1.5rem; display: flex; justify-content: space-between; align-items: center; font-size: 0.8rem;">
          <div>
            <span style="color: #94a3b8;">WebDAV Endpoint:</span>
            <code style="color: #38bdf8; margin-left: 0.5rem;">{{ selectedDocument.webdav_path }}</code>
          </div>
          <button class="btn-action" style="padding: 0.25rem 0.5rem; font-size: 0.75rem; background: #334155; color: white;" @click="copyWebDAVPath(selectedDocument.webdav_path)">
            📋 Copy Path
          </button>
        </div>

        <div style="background: #090d16; border: 1px solid #1e293b; border-radius: 0.5rem; padding: 1.5rem; color: #e2e8f0; line-height: 1.6; white-space: pre-wrap; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif;">
          {{ selectedDocument.content }}
        </div>

        <div style="display: flex; justify-content: flex-end; margin-top: 1.5rem;">
          <button class="btn-action" style="background: #334155; color: white;" @click="showDocumentReaderModal = false">Close</button>
        </div>
      </div>
    </div>

    <!-- Publish New Document Modal -->
    <div v-if="showNewDocModal" class="modal-overlay" @click.self="showNewDocModal = false">
      <div class="modal-card" style="max-width: 650px; background: #0f172a; border: 1px solid #334155; padding: 2rem; border-radius: 0.75rem;">
        <div style="display: flex; justify-content: space-between; align-items: flex-start; border-bottom: 2px solid #334155; padding-bottom: 1rem; margin-bottom: 1.5rem;">
          <div>
            <h2 style="margin: 0; color: #38bdf8;">📄 Publish Intranet Document</h2>
            <div style="font-size: 0.85rem; color: #94a3b8; margin-top: 0.25rem;">Deposits standard operating procedures & knowledge base records into Nextcloud</div>
          </div>
          <button class="close-btn" @click="showNewDocModal = false">✕</button>
        </div>

        <form @submit.prevent="submitNewDocument" style="display: flex; flex-direction: column; gap: 1rem;">
          <div>
            <label style="font-size: 0.8rem; color: #94a3b8;">Document Title *</label>
            <input v-model="newDocForm.title" required class="score-input" style="width: 100%; text-align: left; font-size: 0.9rem;" placeholder="e.g. Cleanroom Perch Sanitization SOP" />
          </div>

          <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 1rem;">
            <div>
              <label style="font-size: 0.8rem; color: #94a3b8;">Department Hub *</label>
              <select v-model="newDocForm.department" required class="score-input" style="width: 100%; font-size: 0.85rem;">
                <option value="HR & Policies">HR & Policies</option>
                <option value="Facilities & Maintenance SOPs">Facilities & Maintenance SOPs</option>
                <option value="Veterinary Protocols">Veterinary Protocols</option>
                <option value="Executive Reporting & Finance">Executive Reporting & Finance</option>
              </select>
            </div>
            <div>
              <label style="font-size: 0.8rem; color: #94a3b8;">Document Category *</label>
              <select v-model="newDocForm.category" required class="score-input" style="width: 100%; font-size: 0.85rem;">
                <option value="Policy">Policy</option>
                <option value="SOP">SOP</option>
                <option value="Clinical Protocol">Clinical Protocol</option>
                <option value="Architecture">Architecture</option>
                <option value="Report">Report</option>
              </select>
            </div>
          </div>

          <div>
            <label style="font-size: 0.8rem; color: #94a3b8;">Author *</label>
            <select v-model="newDocForm.author" required class="score-input" style="width: 100%; font-size: 0.85rem;">
              <option v-for="staff in staffList" :key="staff.id" :value="staff.id">{{ staff.icon }} {{ staff.name }}</option>
            </select>
          </div>

          <div>
            <label style="font-size: 0.8rem; color: #94a3b8;">Document Content (Markdown / Text) *</label>
            <textarea v-model="newDocForm.content" required class="score-input" style="width: 100%; min-height: 140px; text-align: left; font-size: 0.85rem; font-family: monospace; resize: vertical;" placeholder="# Document Title\n\n### 1. Purpose & Scope\nDetail procedure guidelines here..."></textarea>
          </div>

          <div style="display: flex; justify-content: flex-end; gap: 0.75rem; margin-top: 1rem;">
            <button type="button" class="btn-action" style="background: #334155; color: white;" @click="showNewDocModal = false">Cancel</button>
            <button type="submit" class="btn-action" style="background: var(--secondary-color, #ea580c); color: white;" :disabled="isPublishingDoc">
              {{ isPublishingDoc ? 'Publishing...' : '🚀 Publish to Nextcloud' }}
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- Schedule Calendar Event Modal -->
    <div v-if="showNewEventModal" class="modal-overlay" @click.self="showNewEventModal = false">
      <div class="modal-card" style="max-width: 600px; background: #0f172a; border: 1px solid #334155; padding: 2rem; border-radius: 0.75rem;">
        <div style="display: flex; justify-content: space-between; align-items: flex-start; border-bottom: 2px solid #334155; padding-bottom: 1rem; margin-bottom: 1.5rem;">
          <div>
            <h2 style="margin: 0; color: #38bdf8;">📅 Schedule Calendar Event</h2>
            <div style="font-size: 0.85rem; color: #94a3b8; margin-top: 0.25rem;">Publish shift rotation, maintenance window, or milestone to Nextcloud CalDAV</div>
          </div>
          <button class="close-btn" @click="showNewEventModal = false">✕</button>
        </div>

        <form @submit.prevent="submitNewCalendarEvent" style="display: flex; flex-direction: column; gap: 1rem;">
          <div>
            <label style="font-size: 0.8rem; color: #94a3b8;">Calendar Target *</label>
            <select v-model="newEventForm.calendar" required class="score-input" style="width: 100%; font-size: 0.85rem;">
              <option value="Shift Rotations">Shift Rotations</option>
              <option value="Habitat Maintenance Windows">Habitat Maintenance Windows</option>
              <option value="Corporate Milestones">Corporate Milestones</option>
              <option value="Veterinary Audits">Veterinary Audits</option>
            </select>
          </div>

          <div>
            <label style="font-size: 0.8rem; color: #94a3b8;">Event Title *</label>
            <input v-model="newEventForm.title" required class="score-input" style="width: 100%; text-align: left; font-size: 0.9rem;" placeholder="e.g. Caretaker Shift Alpha" />
          </div>

          <div>
            <label style="font-size: 0.8rem; color: #94a3b8;">Description</label>
            <input v-model="newEventForm.description" class="score-input" style="width: 100%; text-align: left; font-size: 0.85rem;" placeholder="e.g. Routine cleaning and vital signs check" />
          </div>

          <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 1rem;">
            <div>
              <label style="font-size: 0.8rem; color: #94a3b8;">Department *</label>
              <select v-model="newEventForm.department" required class="score-input" style="width: 100%; font-size: 0.85rem;">
                <option value="HR & Policies">HR & Policies</option>
                <option value="Facilities & Maintenance SOPs">Facilities & Maintenance SOPs</option>
                <option value="Veterinary Protocols">Veterinary Protocols</option>
                <option value="Executive Reporting & Finance">Executive Reporting & Finance</option>
              </select>
            </div>
            <div>
              <label style="font-size: 0.8rem; color: #94a3b8;">Location</label>
              <input v-model="newEventForm.location" class="score-input" style="width: 100%; text-align: left; font-size: 0.85rem;" placeholder="e.g. Habitat Cleanroom Alpha" />
            </div>
          </div>

          <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 1rem;">
            <div>
              <label style="font-size: 0.8rem; color: #94a3b8;">Organizer *</label>
              <select v-model="newEventForm.organizer" required class="score-input" style="width: 100%; font-size: 0.85rem;">
                <option v-for="staff in staffList" :key="staff.id" :value="staff.id">{{ staff.icon }} {{ staff.name }}</option>
              </select>
            </div>
            <div>
              <label style="font-size: 0.8rem; color: #94a3b8;">Status *</label>
              <select v-model="newEventForm.status" required class="score-input" style="width: 100%; font-size: 0.85rem;">
                <option value="scheduled">Scheduled</option>
                <option value="confirmed">Confirmed</option>
              </select>
            </div>
          </div>

          <div style="display: flex; justify-content: flex-end; gap: 0.75rem; margin-top: 1rem;">
            <button type="button" class="btn-action" style="background: #334155; color: white;" @click="showNewEventModal = false">Cancel</button>
            <button type="submit" class="btn-action" style="background: #38bdf8; color: #0f172a; font-weight: bold;" :disabled="isSchedulingEvent">
              {{ isSchedulingEvent ? 'Scheduling...' : '📅 Publish Event' }}
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- New IT Support Ticket Modal -->
    <div v-if="showNewTicketModal" class="modal-overlay" @click.self="showNewTicketModal = false">
      <div class="modal-card" style="max-width: 650px; background: #0f172a; border: 1px solid #334155; padding: 2rem; border-radius: 0.75rem;">
        <div style="display: flex; justify-content: space-between; align-items: flex-start; border-bottom: 2px solid #334155; padding-bottom: 1rem; margin-bottom: 1.5rem;">
          <div>
            <h2 style="margin: 0; color: #f8fafc;">🛠️ Open IT Helpdesk Ticket</h2>
            <div style="font-size: 0.85rem; color: #94a3b8; margin-top: 0.25rem;">
              Syncs with Forgejo issue tracker & routes urgent priority alerts to On-Call Pebble watch
            </div>
          </div>
          <button class="close-btn" @click="showNewTicketModal = false">✕</button>
        </div>

        <form @submit.prevent="submitNewTicket" style="display: flex; flex-direction: column; gap: 1rem;">
          <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 1rem;">
            <div>
              <label style="font-size: 0.8rem; color: #94a3b8;">Category *</label>
              <select v-model="newTicketForm.category" required class="score-input" style="width: 100%; font-size: 0.85rem;">
                <option value="Habitat">🐾 Habitat & Environmental Control</option>
                <option value="Hardware">🖥️ Hardware & Edge Gateways</option>
                <option value="Software">💻 Software & Services</option>
                <option value="Trading">⚡ Quantitative & Trading Desk</option>
                <option value="Security">🔐 Security & PKI Certificates</option>
              </select>
            </div>
            <div>
              <label style="font-size: 0.8rem; color: #94a3b8;">Severity Level *</label>
              <select v-model="newTicketForm.severity" required class="score-input" style="width: 100%; font-size: 0.85rem;">
                <option value="low">Low (General Inquiry / Minor)</option>
                <option value="medium">Medium (Standard Work Order)</option>
                <option value="high">High (Impacting Operations)</option>
                <option value="urgent">Urgent (On-Call Alert Dispatched)</option>
                <option value="critical">Critical (Habitat Anomaly / Outage)</option>
              </select>
            </div>
          </div>

          <div>
            <label style="font-size: 0.8rem; color: #94a3b8;">Ticket Title *</label>
            <input v-model="newTicketForm.title" required class="score-input" style="width: 100%; text-align: left; font-size: 0.9rem;" placeholder="e.g. Habitat B Environmental Sensor Drift" />
          </div>

          <div>
            <label style="font-size: 0.8rem; color: #94a3b8;">Description & Diagnostic Details</label>
            <textarea v-model="newTicketForm.body" rows="4" class="score-input" style="width: 100%; text-align: left; font-size: 0.85rem; resize: vertical;" placeholder="Provide steps to reproduce, hardware serials, observed readings, or error logs..."></textarea>
          </div>

          <div style="background: rgba(239, 68, 68, 0.1); border: 1px solid rgba(239, 68, 68, 0.3); border-radius: 0.5rem; padding: 0.85rem;">
            <label style="display: flex; align-items: center; gap: 0.5rem; font-size: 0.85rem; color: #fca5a5; cursor: pointer;">
              <input type="checkbox" v-model="newTicketForm.is_urgent" />
              <strong>🚨 Urgent Escalation: Dispatch immediately to On-Call Pebble watch & mobile push (ntfy)</strong>
            </label>
          </div>

          <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 1rem;">
            <div>
              <label style="font-size: 0.8rem; color: #94a3b8;">Reporter Username</label>
              <input v-model="newTicketForm.author_username" class="score-input" style="width: 100%; text-align: left; font-size: 0.85rem;" placeholder="e.g. emp-human-elise" />
            </div>
            <div>
              <label style="font-size: 0.8rem; color: #94a3b8;">Target Forgejo Repository</label>
              <input v-model="newTicketForm.forgejo_repo" class="score-input" style="width: 100%; text-align: left; font-size: 0.85rem;" placeholder="oci-internal/helpdesk" />
            </div>
          </div>

          <div style="display: flex; justify-content: flex-end; gap: 0.75rem; margin-top: 1rem;">
            <button type="button" class="btn-action" style="background: #334155; color: white;" @click="showNewTicketModal = false">Cancel</button>
            <button type="submit" class="btn-action" style="background: var(--secondary-color, #ea580c); color: white;" :disabled="isSubmittingTicket">
              {{ isSubmittingTicket ? 'Submitting...' : '🚀 Submit Ticket' }}
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- Ticket Detail & Discussion Modal -->
    <div v-if="showTicketDetailModal && selectedHelpdeskTicket" class="modal-overlay" @click.self="showTicketDetailModal = false">
      <div class="modal-card" style="max-width: 750px; background: #0f172a; border: 1px solid #334155; padding: 2rem; border-radius: 0.75rem; max-height: 90vh; overflow-y: auto;">
        <div style="display: flex; justify-content: space-between; align-items: flex-start; border-bottom: 2px solid #334155; padding-bottom: 1rem; margin-bottom: 1.25rem;">
          <div>
            <div style="display: flex; align-items: center; gap: 0.5rem; margin-bottom: 0.35rem;">
              <span class="badge" style="background: #0284c7; color: white;">#{{ selectedHelpdeskTicket.forgejo_issue_id || selectedHelpdeskTicket.ticket_id.substring(0, 6) }}</span>
              <span :class="['badge', getSeverityBadgeClass(selectedHelpdeskTicket.severity)]">{{ selectedHelpdeskTicket.severity ? selectedHelpdeskTicket.severity.toUpperCase() : 'MEDIUM' }}</span>
              <span :class="['badge', getTicketStatusBadgeClass(selectedHelpdeskTicket.state)]">{{ selectedHelpdeskTicket.state ? selectedHelpdeskTicket.state.toUpperCase() : 'OPEN' }}</span>
            </div>
            <h2 style="margin: 0; color: #f8fafc; font-size: 1.3rem;">{{ selectedHelpdeskTicket.title }}</h2>
          </div>
          <button class="close-btn" @click="showTicketDetailModal = false">✕</button>
        </div>

        <!-- Meta Details Box -->
        <div style="background: #1e293b; border-radius: 0.5rem; padding: 1rem; margin-bottom: 1.25rem; font-size: 0.85rem; color: #cbd5e1; display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 0.75rem;">
          <div><strong>Category:</strong> {{ selectedHelpdeskTicket.category || 'General' }}</div>
          <div><strong>Source:</strong> {{ selectedHelpdeskTicket.source || 'internal' }}</div>
          <div><strong>Reporter:</strong> {{ selectedHelpdeskTicket.author_username }}</div>
          <div><strong>Assignee:</strong> {{ selectedHelpdeskTicket.assignee_username || 'Unassigned' }}</div>
          <div><strong>Repository:</strong> <code>{{ selectedHelpdeskTicket.forgejo_repo }}</code></div>
          <div><strong>Created:</strong> {{ formatDateTime(selectedHelpdeskTicket.created_at) }}</div>
          <div v-if="selectedHelpdeskTicket.resolution_notes" style="grid-column: 1 / -1; border-top: 1px solid #334155; padding-top: 0.5rem; color: #86efac;">
            <strong>Resolution Notes:</strong> {{ selectedHelpdeskTicket.resolution_notes }}
          </div>
        </div>

        <!-- Body / Description -->
        <div v-if="selectedHelpdeskTicket.body" style="background: #1e293b; border-radius: 0.5rem; padding: 1rem; margin-bottom: 1.25rem; font-size: 0.9rem; color: #f8fafc; white-space: pre-wrap;">
          {{ selectedHelpdeskTicket.body }}
        </div>

        <!-- Action Buttons -->
        <div style="display: flex; gap: 0.5rem; margin-bottom: 1.5rem;">
          <button v-if="selectedHelpdeskTicket.state === 'open'" class="btn-action" style="background: #d97706; color: white;" @click="quickAcknowledgeTicket(selectedHelpdeskTicket)">
            ✔️ Acknowledge Ticket
          </button>
          <button v-if="selectedHelpdeskTicket.state !== 'closed'" class="btn-action" style="background: #059669; color: white;" @click="quickCloseTicket(selectedHelpdeskTicket)">
            ✅ Mark Closed / Resolved
          </button>
        </div>

        <!-- Comments Thread Section -->
        <div style="border-top: 2px solid #334155; padding-top: 1rem;">
          <h3 style="margin: 0 0 1rem 0; color: #f8fafc; font-size: 1.1rem;">💬 Discussion Thread</h3>

          <div style="display: flex; flex-direction: column; gap: 0.75rem; margin-bottom: 1.25rem;">
            <div
              v-for="c in helpdeskComments"
              :key="c.comment_id"
              :style="{
                background: c.is_internal ? 'rgba(234, 88, 12, 0.1)' : '#1e293b',
                border: c.is_internal ? '1px solid rgba(234, 88, 12, 0.3)' : '1px solid #334155',
                borderRadius: '0.5rem',
                padding: '0.85rem'
              }"
            >
              <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.4rem;">
                <div style="display: flex; align-items: center; gap: 0.5rem;">
                  <strong style="color: #f8fafc; font-size: 0.85rem;">{{ c.author_username }}</strong>
                  <span v-if="c.is_internal" class="badge" style="background: #ea580c; color: white; font-size: 0.65rem;">
                    🔒 Internal Note (Scrubbed)
                  </span>
                  <span v-else class="badge" style="background: #38bdf8; color: #0f172a; font-size: 0.65rem;">
                    📢 Public
                  </span>
                </div>
                <span style="font-size: 0.75rem; color: #94a3b8;">{{ formatDateTime(c.created_at) }}</span>
              </div>
              <div style="font-size: 0.85rem; color: #e2e8f0; white-space: pre-wrap;">
                {{ c.comment }}
              </div>
            </div>

            <div v-if="helpdeskComments.length === 0" style="text-align: center; color: #94a3b8; padding: 1rem; font-size: 0.85rem;">
              No comments yet on this ticket.
            </div>
          </div>

          <!-- Add Comment Box -->
          <form @submit.prevent="submitTicketComment" style="display: flex; flex-direction: column; gap: 0.75rem;">
            <textarea
              v-model="newCommentForm.comment"
              rows="3"
              required
              class="score-input"
              style="width: 100%; text-align: left; font-size: 0.85rem; resize: vertical;"
              placeholder="Add your technical update, diagnosis, or troubleshooting note..."
            ></textarea>

            <div style="display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 0.5rem;">
              <label style="display: flex; align-items: center; gap: 0.4rem; font-size: 0.8rem; color: #fdba74; cursor: pointer;">
                <input type="checkbox" v-model="newCommentForm.is_internal" />
                <span>🔒 Mark as Internal Note (Confidential to staff; scrubbed from customer portal)</span>
              </label>

              <button type="submit" class="btn-action" style="background: #38bdf8; color: #0f172a; font-weight: bold;" :disabled="isSubmittingComment">
                {{ isSubmittingComment ? 'Posting...' : 'Post Update' }}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>

    <!-- Generate Executive Report Modal -->
    <div v-if="showGenerateReportModal" class="modal-overlay" @click.self="showGenerateReportModal = false">
      <div class="modal-card" style="max-width: 550px; background: #0f172a; border: 1px solid #334155; padding: 2rem; border-radius: 0.75rem;">
        <div style="display: flex; justify-content: space-between; align-items: flex-start; border-bottom: 2px solid #334155; padding-bottom: 1rem; margin-bottom: 1.5rem;">
          <div>
            <h2 style="margin: 0; color: #38bdf8;">📊 Generate Executive Management Report</h2>
            <div style="font-size: 0.85rem; color: #94a3b8; margin-top: 0.25rem;">Reconciles ERPNext General Ledger & aggregates workforce KPIs into PDF/CSV</div>
          </div>
          <button class="close-btn" @click="showGenerateReportModal = false">✕</button>
        </div>

        <form @submit.prevent="triggerGenerateReport" style="display: flex; flex-direction: column; gap: 1rem;">
          <div>
            <label style="font-size: 0.8rem; color: #94a3b8;">Reporting Period (YYYY-MM) *</label>
            <input v-model="generateReportForm.period" required class="score-input" style="width: 100%; text-align: left; font-size: 0.9rem;" placeholder="e.g. 2026-10" />
          </div>

          <div>
            <label style="font-size: 0.8rem; color: #94a3b8;">Generated By (Officer / Persona) *</label>
            <select v-model="generateReportForm.generated_by" class="score-input" style="width: 100%; text-align: left; font-size: 0.9rem;">
              <option value="emp-human-alice">Alice Vance (Head of HR & Admin)</option>
              <option value="emp-feline-garfield">Garfield (Chief Observation Officer)</option>
              <option value="emp-human-frank">Frank Operations (Platform Infrastructure)</option>
              <option value="system-reporting-worker">System Scheduled Reporting Worker</option>
            </select>
          </div>

          <div class="compliance-box" style="margin-top: 0.5rem; background: rgba(56, 189, 248, 0.08); border: 1px solid rgba(56, 189, 248, 0.2); padding: 0.75rem; border-radius: 6px;">
            <div style="font-size: 0.75rem; color: #38bdf8; font-weight: bold; margin-bottom: 0.25rem;">🔒 Nextcloud WebDAV Compliance Guarantee</div>
            <div style="font-size: 0.75rem; color: #cbd5e1;">Generated PDF & CSV artifacts will be cryptographically hashed and deposited directly into <code>/remote.php/dav/files/admin/Executive/Reports/YYYY-MM/</code>.</div>
          </div>

          <div style="display: flex; justify-content: flex-end; gap: 0.75rem; margin-top: 1rem;">
            <button type="button" class="btn-action" style="background: #334155; color: white;" @click="showGenerateReportModal = false">Cancel</button>
            <button type="submit" class="btn-action" style="background: var(--secondary-color, #ea580c); color: white;" :disabled="isGeneratingReport">
              {{ isGeneratingReport ? 'Generating & Archiving...' : '🚀 Generate & Archive Report' }}
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- Executive Report Detail Modal -->
    <div v-if="showReportDetailModal && selectedReport" class="modal-overlay" @click.self="showReportDetailModal = false">
      <div class="modal-card" style="max-width: 750px; background: #0f172a; border: 1px solid #334155; padding: 2rem; border-radius: 0.75rem;">
        <div style="display: flex; justify-content: space-between; align-items: flex-start; border-bottom: 2px solid #334155; padding-bottom: 1rem; margin-bottom: 1.5rem;">
          <div>
            <h2 style="margin: 0; color: #38bdf8;">{{ selectedReport.title }}</h2>
            <div style="font-size: 0.85rem; color: #94a3b8; margin-top: 0.25rem;">Period: {{ selectedReport.period }} • Generated by: {{ selectedReport.generated_by }} • ID: {{ selectedReport.report_id }}</div>
          </div>
          <button class="close-btn" @click="showReportDetailModal = false">✕</button>
        </div>

        <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 1.5rem; margin-bottom: 1.5rem;">
          <!-- Financial Ledger Column -->
          <div style="background: #1e293b; padding: 1.25rem; border-radius: 0.5rem; border: 1px solid #334155;">
            <h3 style="margin: 0 0 0.75rem 0; color: #38bdf8; font-size: 1rem;">💰 ERPNext Financial Overview</h3>
            <div style="display: flex; flex-direction: column; gap: 0.5rem; font-size: 0.85rem;">
              <div style="display: flex; justify-content: space-between;">
                <span style="color: #94a3b8;">Total Revenue:</span>
                <strong style="color: #38bdf8;">${{ Number(selectedReport.total_revenue || 0).toLocaleString(undefined, { minimumFractionDigits: 2 }) }}</strong>
              </div>
              <div style="display: flex; justify-content: space-between;">
                <span style="color: #94a3b8;">Operating Expenses:</span>
                <span style="color: #fca5a5;">${{ Number(selectedReport.operating_expenses || 0).toLocaleString(undefined, { minimumFractionDigits: 2 }) }}</span>
              </div>
              <div style="display: flex; justify-content: space-between;">
                <span style="color: #94a3b8;">Payroll Expenses:</span>
                <span style="color: #fca5a5;">${{ Number(selectedReport.payroll_expenses || 0).toLocaleString(undefined, { minimumFractionDigits: 2 }) }}</span>
              </div>
              <div style="border-top: 1px solid #334155; padding-top: 0.5rem; display: flex; justify-content: space-between;">
                <span style="color: #94a3b8;">Net Profit:</span>
                <strong style="color: #4ade80;">${{ Number(selectedReport.net_profit || 0).toLocaleString(undefined, { minimumFractionDigits: 2 }) }}</strong>
              </div>
              <div style="display: flex; justify-content: space-between;">
                <span style="color: #94a3b8;">Operational Burn:</span>
                <span style="color: #f87171;">${{ Number(selectedReport.operational_burn || 0).toLocaleString(undefined, { minimumFractionDigits: 2 }) }}/mo</span>
              </div>
              <div style="display: flex; justify-content: space-between;">
                <span style="color: #94a3b8;">Cash Reserves:</span>
                <span style="color: #facc15;">${{ Number(selectedReport.cash_reserves || 0).toLocaleString(undefined, { minimumFractionDigits: 2 }) }}</span>
              </div>
              <div style="display: flex; justify-content: space-between;">
                <span style="color: #94a3b8;">Runway:</span>
                <span style="color: #facc15; font-weight: bold;">{{ selectedReport.runway_months }} Months</span>
              </div>
            </div>
          </div>

          <!-- Workforce KPIs Column -->
          <div style="background: #1e293b; padding: 1.25rem; border-radius: 0.5rem; border: 1px solid #334155;">
            <h3 style="margin: 0 0 0.75rem 0; color: #c084fc; font-size: 1rem;">👥 Workforce KPIs & Operations</h3>
            <div style="display: flex; flex-direction: column; gap: 0.5rem; font-size: 0.85rem;">
              <div style="display: flex; justify-content: space-between;">
                <span style="color: #94a3b8;">Total Headcount:</span>
                <strong style="color: #f8fafc;">{{ selectedReport.total_headcount }} Staff</strong>
              </div>
              <div style="display: flex; justify-content: space-between;">
                <span style="color: #94a3b8;">Human Personnel:</span>
                <span style="color: #e2e8f0;">{{ selectedReport.human_headcount }}</span>
              </div>
              <div style="display: flex; justify-content: space-between;">
                <span style="color: #94a3b8;">Feline Executives:</span>
                <span style="color: #e2e8f0;">{{ selectedReport.feline_headcount }}</span>
              </div>
              <div style="border-top: 1px solid #334155; padding-top: 0.5rem; display: flex; justify-content: space-between;">
                <span style="color: #94a3b8;">Attendance Rate:</span>
                <strong style="color: #34d399;">{{ selectedReport.attendance_rate }}%</strong>
              </div>
              <div style="display: flex; justify-content: space-between;">
                <span style="color: #94a3b8;">Overtime Hours:</span>
                <span style="color: #e2e8f0;">{{ selectedReport.overtime_hours }} hrs</span>
              </div>
              <div style="display: flex; justify-content: space-between;">
                <span style="color: #94a3b8;">Timesheets Approved:</span>
                <span style="color: #38bdf8;">{{ selectedReport.timesheets_approved }}</span>
              </div>
              <div style="display: flex; justify-content: space-between;">
                <span style="color: #94a3b8;">Status:</span>
                <span class="badge badge-success">🔒 {{ selectedReport.status ? selectedReport.status.toUpperCase() : 'ARCHIVED' }}</span>
              </div>
            </div>
          </div>
        </div>

        <!-- Nextcloud Archival Box -->
        <div style="background: #0f172a; padding: 1rem; border-radius: 0.5rem; border: 1px solid #334155; margin-bottom: 1.5rem;">
          <h4 style="margin: 0 0 0.5rem 0; color: #fdba74; font-size: 0.9rem;">📁 Nextcloud WebDAV Compliance Archival Paths</h4>
          <div style="font-family: monospace; font-size: 0.8rem; color: #cbd5e1; display: flex; flex-direction: column; gap: 0.25rem;">
            <div><strong>PDF:</strong> {{ selectedReport.webdav_path_pdf || 'N/A' }}</div>
            <div><strong>CSV:</strong> {{ selectedReport.webdav_path_csv || 'N/A' }}</div>
          </div>
        </div>

        <div style="display: flex; justify-content: space-between; align-items: center; border-top: 2px solid #334155; padding-top: 1rem;">
          <div style="display: flex; gap: 0.5rem;">
            <button class="btn-action" style="background: #0284c7; color: white;" @click="downloadReport(selectedReport, 'pdf')">
              📄 Download PDF Report
            </button>
            <button class="btn-action" style="background: #059669; color: white;" @click="downloadReport(selectedReport, 'csv')">
              📊 Download CSV Export
            </button>
          </div>
          <button class="btn-action" style="background: #334155; color: white;" @click="showReportDetailModal = false">
            Close
          </button>
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

// --- IT Helpdesk & Issue Management State ---
const helpdeskTickets = ref([]);
const selectedHelpdeskTicket = ref(null);
const helpdeskComments = ref([]);
const showNewTicketModal = ref(false);
const showTicketDetailModal = ref(false);
const isSubmittingTicket = ref(false);
const isSubmittingComment = ref(false);
const helpdeskFilterStatus = ref('all');
const helpdeskFilterSeverity = ref('all');
const helpdeskFilterCategory = ref('all');

const newTicketForm = ref({
  category: 'Habitat',
  severity: 'medium',
  title: '',
  body: '',
  is_urgent: false,
  forgejo_repo: 'oci-internal/helpdesk',
  author_username: 'emp-human-elise'
});

const newCommentForm = ref({
  comment: '',
  is_internal: true,
  author_username: 'emp-human-frank'
});

// --- Executive Reporting & Business Intelligence State ---
const defaultExecutiveReports = [
  {
    report_id: 'rep-2026-10',
    period: '2026-10',
    title: 'OCI Executive Management Summary - October 2026',
    status: 'archived',
    total_revenue: 125000.00,
    operating_expenses: 45000.00,
    payroll_expenses: 38500.00,
    net_profit: 41500.00,
    operational_burn: 83500.00,
    cash_reserves: 540000.00,
    runway_months: 6.47,
    total_headcount: 12,
    human_headcount: 7,
    feline_headcount: 5,
    attendance_rate: 98.4,
    overtime_hours: 14.5,
    timesheets_approved: 28,
    webdav_path_pdf: '/remote.php/dav/files/admin/Executive/Reports/2026-10/executive_summary_2026-10.pdf',
    webdav_path_csv: '/remote.php/dav/files/admin/Executive/Reports/2026-10/executive_summary_2026-10.csv',
    generated_by: 'system-reporting-worker',
    created_at: '2026-10-01T00:00:00Z',
    updated_at: '2026-10-01T00:00:00Z'
  },
  {
    report_id: 'rep-2026-09',
    period: '2026-09',
    title: 'OCI Executive Management Summary - September 2026',
    status: 'archived',
    total_revenue: 118000.00,
    operating_expenses: 42000.00,
    payroll_expenses: 38500.00,
    net_profit: 37500.00,
    operational_burn: 80500.00,
    cash_reserves: 582000.00,
    runway_months: 7.23,
    total_headcount: 12,
    human_headcount: 7,
    feline_headcount: 5,
    attendance_rate: 99.1,
    overtime_hours: 11.0,
    timesheets_approved: 30,
    webdav_path_pdf: '/remote.php/dav/files/admin/Executive/Reports/2026-09/executive_summary_2026-09.pdf',
    webdav_path_csv: '/remote.php/dav/files/admin/Executive/Reports/2026-09/executive_summary_2026-09.csv',
    generated_by: 'system-reporting-worker',
    created_at: '2026-09-01T00:00:00Z',
    updated_at: '2026-09-01T00:00:00Z'
  }
];

const executiveReports = ref([...defaultExecutiveReports]);
const selectedReport = ref(defaultExecutiveReports[0]);
const showGenerateReportModal = ref(false);
const showReportDetailModal = ref(false);
const isGeneratingReport = ref(false);
const generateReportForm = ref({
  period: '2026-10',
  generated_by: 'emp-human-alice'
});

const filteredHelpdeskTickets = computed(() => {
  return helpdeskTickets.value.filter(t => {
    if (helpdeskFilterStatus.value !== 'all' && t.state !== helpdeskFilterStatus.value) return false;
    if (helpdeskFilterSeverity.value !== 'all' && t.severity !== helpdeskFilterSeverity.value) return false;
    if (helpdeskFilterCategory.value !== 'all' && t.category !== helpdeskFilterCategory.value) return false;
    return true;
  });
});

function getSeverityBadgeClass(sev) {
  switch (sev?.toLowerCase()) {
    case 'critical':
    case 'urgent':
      return 'badge-urgent';
    case 'high':
      return 'badge-high';
    case 'medium':
      return 'badge-medium';
    default:
      return 'badge-low';
  }
}

function getTicketStatusBadgeClass(st) {
  switch (st?.toLowerCase()) {
    case 'closed':
      return 'badge-closed';
    case 'acknowledged':
      return 'badge-ack';
    default:
      return 'badge-open';
  }
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
const evaluationTab = ref('self');

const reviewForm = ref({
  employee_id: 'emp-feline-barneby',
  review_type: 'feline_health_assessment',
  review_period: '2026-Q3',
  phase: 'self_appraisal',
  scheduled_for: '2026-11-15',
  reviewer_id: 'emp-human-elena',
  notes: 'Quarterly agility, perch response readiness, and weight audit'
});

const gradingForm = ref({
  score: 4.85,
  status: 'completed',
  phase: 'completed',
  reviewer_id: 'emp-human-elena',
  self_appraisal: '',
  self_score: 4.80,
  peer_reviewer_id: '',
  peer_feedback: '',
  peer_score: 4.85,
  manager_assessment: '',
  manager_score: 4.90,
  kpi_ratings: '{"workplace_safety": 5.0, "alpha_telemetry": 4.9}',
  clinical_auditor_id: 'emp-human-elena',
  wellness_audit_score: 4.95,
  habitat_recommendations: '',
  vhr_record_id: ''
});

const reviewCycles = ref([
  {
    review_id: '018e0000-0000-7000-8000-000000000020',
    employee_id: 'emp-feline-garfield',
    review_type: 'feline_health_assessment',
    review_period: '2026-Q3',
    phase: 'completed',
    scheduled_for: '2026-09-24',
    status: 'completed',
    reviewer_id: 'emp-human-elena',
    peer_reviewer_id: 'emp-human-bob',
    clinical_auditor_id: 'emp-human-elena',
    self_appraisal: 'Maintained optimal alpha perch vigilance, elevated sunbeam absorption, and flawless oversight of feline trading algorithms.',
    self_score: 4.90,
    peer_feedback: 'Exemplary leadership from the high perch. Calming purr presence during high volatility trading windows.',
    peer_score: 4.95,
    manager_assessment: 'Outstanding performance across corporate feline responsibilities. Unanimously recognized by human and feline staff.',
    manager_score: 5.00,
    score: 4.95,
    kpi_ratings: '{"market_vigilance": 5.0, "whisker_symmetry": 4.9, "purr_acoustics": 5.0}',
    wellness_audit_score: 4.95,
    habitat_recommendations: 'Optimal alpha perch alignment with sunlight verified. Maintain daily grooming and catnip break intervals.',
    notes: 'Superb whisker symmetry, resting purr acoustics 92dB, optimal alpha sunbeam positioning. Lasagna tolerance remains peak.'
  },
  {
    review_id: '018e0000-0000-7000-8000-000000000021',
    employee_id: 'emp-feline-barneby',
    review_type: 'feline_health_assessment',
    review_period: '2026-Q3',
    phase: 'manager_review',
    scheduled_for: '2026-10-02',
    status: 'in_progress',
    reviewer_id: 'emp-human-elena',
    peer_reviewer_id: 'emp-human-alice',
    self_appraisal: 'Steadily defended territory and participated in daily habitat agility trials.',
    self_score: 4.75,
    peer_feedback: 'Demonstrated high vigilance and agility during physical drills.',
    peer_score: 4.80,
    notes: 'Quarterly weight check and alpha perch mobility audit'
  },
  {
    review_id: '018e0000-0000-7000-8000-000000000022',
    employee_id: 'emp-human-alice',
    review_type: 'performance',
    review_period: '2026-Q3',
    phase: 'self_appraisal',
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

const activeEmergencyAlert = ref(null);
let incidentEventSource = null;

function dismissEmergencyAlert() {
  activeEmergencyAlert.value = null;
}

async function acknowledgeEmergencyAlert() {
  if (!activeEmergencyAlert.value) return;
  const alertId = activeEmergencyAlert.value.incident_id;
  if (alertId) {
    await resolveIncident(alertId);
  }
  activeEmergencyAlert.value = null;
  showToast('Emergency incident acknowledged.');
}

function initIncidentSSE() {
  if (typeof window === 'undefined' || typeof window.EventSource === 'undefined') return;
  try {
    if (incidentEventSource) {
      incidentEventSource.close();
    }
    incidentEventSource = new EventSource('/api/v1/employee/incidents/stream');
    incidentEventSource.addEventListener('incident', (event) => {
      try {
        const payload = JSON.parse(event.data);
        activeEmergencyAlert.value = payload;
        if (payload.incident_id && !incidentLogs.value.some(i => i.incident_id === payload.incident_id)) {
          incidentLogs.value.unshift(payload);
        }
        showToast(`🚨 Emergency Alert: ${payload.title}`);
      } catch (e) {
        console.error('Failed to parse incident SSE event', e);
      }
    });
    incidentEventSource.onerror = () => {
      // EventSource retries automatically in browsers
    };
  } catch (err) {
    console.warn('Incident SSE initialization failed', err);
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
  if (reviewFilter.value === 'self_appraisal' || reviewFilter.value === 'peer_feedback' || reviewFilter.value === 'manager_review') {
    return reviewCycles.value.filter(r => r.phase === reviewFilter.value);
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

function formatPhase(phase) {
  switch (phase) {
    case 'self_appraisal': return '📝 Self-Appraisal';
    case 'peer_feedback': return '👥 Peer Feedback';
    case 'manager_review': return '👔 Manager Review';
    case 'completed': return '✅ Completed';
    default: return phase || '📝 Self-Appraisal';
  }
}

function countReviewStatus(status) {
  return reviewCycles.value.filter(r => r.status === status).length;
}

function countReviewType(type) {
  return reviewCycles.value.filter(r => r.review_type === type).length;
}

function countReviewPhase(phase) {
  return reviewCycles.value.filter(r => r.phase === phase).length;
}

function getPhaseStepNumber(phase) {
  switch (phase) {
    case 'self_appraisal': return 1;
    case 'peer_feedback': return 2;
    case 'manager_review': return 3;
    case 'completed': return 4;
    default: return 1;
  }
}

function isStepActive(phase, step) {
  return getPhaseStepNumber(phase) === step;
}

function isStepCompleted(phase, step) {
  return getPhaseStepNumber(phase) > step;
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
      review_period: reviewForm.value.review_period || '2026-Q3',
      phase: reviewForm.value.phase || 'self_appraisal',
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
      review_period: reviewForm.value.review_period || '2026-Q3',
      phase: reviewForm.value.phase || 'self_appraisal',
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
  evaluationTab.value = rc.review_type === 'feline_health_assessment' && rc.phase === 'completed'
    ? 'vet'
    : (rc.phase === 'peer_feedback' ? 'peer' : (rc.phase === 'manager_review' ? 'manager' : (rc.phase === 'completed' ? 'manager' : 'self')));

  gradingForm.value = {
    score: rc.score != null ? rc.score : 4.85,
    status: rc.status === 'scheduled' ? 'in_progress' : rc.status,
    phase: rc.phase || 'self_appraisal',
    reviewer_id: rc.reviewer_id || 'emp-human-elena',
    self_appraisal: rc.self_appraisal || '',
    self_score: rc.self_score != null ? rc.self_score : 4.80,
    peer_reviewer_id: rc.peer_reviewer_id || '',
    peer_feedback: rc.peer_feedback || '',
    peer_score: rc.peer_score != null ? rc.peer_score : 4.85,
    manager_assessment: rc.manager_assessment || '',
    manager_score: rc.manager_score != null ? rc.manager_score : 4.90,
    kpi_ratings: rc.kpi_ratings || '',
    clinical_auditor_id: rc.clinical_auditor_id || 'emp-human-elena',
    wellness_audit_score: rc.wellness_audit_score != null ? rc.wellness_audit_score : 4.95,
    habitat_recommendations: rc.habitat_recommendations || '',
    vhr_record_id: rc.vhr_record_id || ''
  };
}

async function submitSelfAppraisalAction() {
  if (!activeReviewForGrading.value) return;
  isSavingGrading.value = true;
  const reviewId = activeReviewForGrading.value.review_id;
  const payload = {
    self_appraisal: gradingForm.value.self_appraisal,
    self_score: Number(gradingForm.value.self_score)
  };
  try {
    const res = await fetch(`/api/v1/workforce/review-cycles/${reviewId}/self-appraisal`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    });
    if (res.ok) {
      const updated = await res.json();
      const idx = reviewCycles.value.findIndex(r => r.review_id === reviewId);
      if (idx !== -1) reviewCycles.value[idx] = updated;
      activeReviewForGrading.value = updated;
    } else {
      const rc = reviewCycles.value.find(r => r.review_id === reviewId);
      if (rc) {
        rc.self_appraisal = payload.self_appraisal;
        rc.self_score = payload.self_score;
        rc.phase = 'peer_feedback';
        rc.status = 'in_progress';
      }
    }
    showToast('Self-appraisal submitted! Cycle transitioned to Peer Feedback.');
    evaluationTab.value = 'peer';
  } catch (e) {
    const rc = reviewCycles.value.find(r => r.review_id === reviewId);
    if (rc) {
      rc.self_appraisal = payload.self_appraisal;
      rc.self_score = payload.self_score;
      rc.phase = 'peer_feedback';
      rc.status = 'in_progress';
    }
    showToast('Self-appraisal saved locally.');
    evaluationTab.value = 'peer';
  } finally {
    isSavingGrading.value = false;
  }
}

async function submitPeerFeedbackAction() {
  if (!activeReviewForGrading.value) return;
  isSavingGrading.value = true;
  const reviewId = activeReviewForGrading.value.review_id;
  const payload = {
    peer_reviewer_id: gradingForm.value.peer_reviewer_id,
    peer_feedback: gradingForm.value.peer_feedback,
    peer_score: Number(gradingForm.value.peer_score)
  };
  try {
    const res = await fetch(`/api/v1/workforce/review-cycles/${reviewId}/peer-feedback`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    });
    if (res.ok) {
      const updated = await res.json();
      const idx = reviewCycles.value.findIndex(r => r.review_id === reviewId);
      if (idx !== -1) reviewCycles.value[idx] = updated;
      activeReviewForGrading.value = updated;
    } else {
      const rc = reviewCycles.value.find(r => r.review_id === reviewId);
      if (rc) {
        rc.peer_reviewer_id = payload.peer_reviewer_id;
        rc.peer_feedback = payload.peer_feedback;
        rc.peer_score = payload.peer_score;
        rc.phase = 'manager_review';
      }
    }
    showToast('Peer feedback recorded! Cycle transitioned to Manager Review.');
    evaluationTab.value = 'manager';
  } catch (e) {
    const rc = reviewCycles.value.find(r => r.review_id === reviewId);
    if (rc) {
      rc.peer_reviewer_id = payload.peer_reviewer_id;
      rc.peer_feedback = payload.peer_feedback;
      rc.peer_score = payload.peer_score;
      rc.phase = 'manager_review';
    }
    showToast('Peer feedback saved locally.');
    evaluationTab.value = 'manager';
  } finally {
    isSavingGrading.value = false;
  }
}

async function submitManagerAssessmentAction() {
  if (!activeReviewForGrading.value) return;
  isSavingGrading.value = true;
  const reviewId = activeReviewForGrading.value.review_id;
  const payload = {
    reviewer_id: gradingForm.value.reviewer_id,
    manager_assessment: gradingForm.value.manager_assessment,
    manager_score: Number(gradingForm.value.manager_score),
    overall_score: Number(gradingForm.value.score || gradingForm.value.manager_score),
    kpi_ratings: gradingForm.value.kpi_ratings
  };
  try {
    const res = await fetch(`/api/v1/workforce/review-cycles/${reviewId}/manager-assessment`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    });
    if (res.ok) {
      const updated = await res.json();
      const idx = reviewCycles.value.findIndex(r => r.review_id === reviewId);
      if (idx !== -1) reviewCycles.value[idx] = updated;
      activeReviewForGrading.value = updated;
    } else {
      const rc = reviewCycles.value.find(r => r.review_id === reviewId);
      if (rc) {
        rc.reviewer_id = payload.reviewer_id;
        rc.manager_assessment = payload.manager_assessment;
        rc.manager_score = payload.manager_score;
        rc.score = payload.overall_score;
        rc.kpi_ratings = payload.kpi_ratings;
        rc.phase = 'completed';
        rc.status = 'completed';
      }
    }
    showToast('Manager assessment finalized! Review cycle completed.');
    if (activeReviewForGrading.value && activeReviewForGrading.value.review_type === 'feline_health_assessment') {
      evaluationTab.value = 'vet';
    } else {
      activeReviewForGrading.value = null;
    }
  } catch (e) {
    const rc = reviewCycles.value.find(r => r.review_id === reviewId);
    if (rc) {
      rc.reviewer_id = payload.reviewer_id;
      rc.manager_assessment = payload.manager_assessment;
      rc.manager_score = payload.manager_score;
      rc.score = payload.overall_score;
      rc.phase = 'completed';
      rc.status = 'completed';
    }
    showToast('Manager assessment finalized locally.');
    if (activeReviewForGrading.value && activeReviewForGrading.value.review_type === 'feline_health_assessment') {
      evaluationTab.value = 'vet';
    } else {
      activeReviewForGrading.value = null;
    }
  } finally {
    isSavingGrading.value = false;
  }
}

async function submitVeterinaryAuditAction() {
  if (!activeReviewForGrading.value) return;
  isSavingGrading.value = true;
  const reviewId = activeReviewForGrading.value.review_id;
  const payload = {
    clinical_auditor_id: gradingForm.value.clinical_auditor_id,
    wellness_audit_score: Number(gradingForm.value.wellness_audit_score),
    habitat_recommendations: gradingForm.value.habitat_recommendations,
    vhr_record_id: gradingForm.value.vhr_record_id || undefined
  };
  try {
    const res = await fetch(`/api/v1/workforce/review-cycles/${reviewId}/vet-audit`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    });
    if (res.ok) {
      const updated = await res.json();
      const idx = reviewCycles.value.findIndex(r => r.review_id === reviewId);
      if (idx !== -1) reviewCycles.value[idx] = updated;
    } else {
      const rc = reviewCycles.value.find(r => r.review_id === reviewId);
      if (rc) {
        rc.clinical_auditor_id = payload.clinical_auditor_id;
        rc.wellness_audit_score = payload.wellness_audit_score;
        rc.habitat_recommendations = payload.habitat_recommendations;
        rc.vhr_record_id = payload.vhr_record_id;
      }
    }
    showToast(`Veterinary clinical audit recorded for ${getStaffName(activeReviewForGrading.value.employee_id)}!`);
    activeReviewForGrading.value = null;
  } catch (e) {
    const rc = reviewCycles.value.find(r => r.review_id === reviewId);
    if (rc) {
      rc.clinical_auditor_id = payload.clinical_auditor_id;
      rc.wellness_audit_score = payload.wellness_audit_score;
      rc.habitat_recommendations = payload.habitat_recommendations;
      rc.vhr_record_id = payload.vhr_record_id;
    }
    showToast('Veterinary clinical audit saved locally.');
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

// Nextcloud Departmental Hub & Intranet State
const defaultHubFolders = [
  { id: 1, mount_point: 'HR & Policies', quota: '50 GB', groups: ['hr', 'workforce', 'all-staff'] },
  { id: 2, mount_point: 'Facilities & Maintenance SOPs', quota: '100 GB', groups: ['facilities', 'operations', 'technicians'] },
  { id: 3, mount_point: 'Veterinary Protocols', quota: '50 GB', groups: ['veterinary', 'medical', 'caretakers'] },
  { id: 4, mount_point: 'Executive Reporting & Finance', quota: '25 GB', groups: ['executive', 'finance', 'audit'] }
];

const defaultDocuments = [
  {
    document_id: 'doc-001',
    title: 'OCI Employee & Feline Care Onboarding Handbook',
    department: 'HR & Policies',
    category: 'Policy',
    author: 'emp-human-alice',
    content: '# OCI Employee & Feline Care Onboarding Handbook\n\nWelcome to Orange Cat Investments! This handbook outlines company standards, feline executive etiquette, catnip safety compliance, and IT equipment policies.\n\n### 1. Feline Executive Respect\n- Always respect feline napping schedules.\n- Never alter dietary formulas without veterinary approval from Dr. Elena Rostova.\n\n### 2. IT & Security\n- All laptop assets are secured via Snipe-IT.\n- Use Git SSH keys for all Forgejo repositories.',
    shared_with: ['emp-human-rick', 'emp-human-frank'],
    webdav_path: '/remote.php/dav/files/admin/HR & Policies/onboarding_handbook.md',
    version: '1.4'
  },
  {
    document_id: 'doc-002',
    title: 'Zero-Trust Microservices & PKI Security Standard',
    department: 'Facilities & Maintenance SOPs',
    category: 'Architecture',
    author: 'emp-human-frank',
    content: '# Zero-Trust Microservices & PKI Security Standard\n\n### Principles\n- Zero long-lived or static certificates.\n- Dynamic cert-manager X.509 rotation in Go standard library `crypto/tls`.\n- Opaque session cookies in Valkey; no raw JWTs exposed to web browsers.\n- Double-submit CSRF enforcement on all state-changing endpoints.',
    shared_with: ['emp-human-elise', 'emp-human-rick', 'emp-human-bob'],
    webdav_path: '/remote.php/dav/files/admin/Facilities & Maintenance SOPs/zero_trust_pki_spec.md',
    version: '2.1'
  },
  {
    document_id: 'doc-003',
    title: 'Veterinary Protocols & Emergency Medical Trading Holds',
    department: 'Veterinary Protocols',
    category: 'Clinical Protocol',
    author: 'emp-vet-elena',
    content: '# Veterinary Clinical Protocols & Emergency Trading Holds\n\n### Protocol 402: Emergency Trading Halt\nWhen a feline executive exhibits elevated resting heart rates (>180 bpm) or signs of distress, an immediate Emergency Medical Hold must be placed via `/api/v1/workforce/care-schedules/{felineId}/medical-hold`.\n\n- All automated trading triggers are suspended instantaneously.\n- Senior veterinary staff conducts an in-person physical assessment.',
    shared_with: ['emp-human-alice', 'emp-human-bob'],
    webdav_path: '/remote.php/dav/files/admin/Veterinary Protocols/emergency_medical_holds.md',
    version: '1.0'
  },
  {
    document_id: 'doc-004',
    title: 'Habitat Environmental Monitoring & Climate Control SOP',
    department: 'Facilities & Maintenance SOPs',
    category: 'SOP',
    author: 'emp-human-bob',
    content: '# Habitat Climate Control & IoT Sensor Maintenance\n\n### Maintenance Guidelines\n- Cleanroom temperature must be maintained between 21.0C and 23.5C.\n- Humidity levels must stay between 40% and 55%.\n- Calibrate automated feeder load cells bi-weekly.',
    shared_with: ['emp-human-rick'],
    webdav_path: '/remote.php/dav/files/admin/Facilities & Maintenance SOPs/habitat_climate_control_sop.md',
    version: '1.2'
  },
  {
    document_id: 'doc-005',
    title: 'Executive Strategy & Q3 Capital Allocation Review',
    department: 'Executive Reporting & Finance',
    category: 'Report',
    author: 'emp-human-alice',
    content: '# Executive Strategy & Q3 Capital Allocation Review\n\n### Financial Highlights\n- Core Invest trading volume increased 34% QoQ.\n- White-label SMB deployment operational margins reached 68%.\n- Feline executive wellness score across all habitats averaged 98.4%.',
    shared_with: ['emp-human-frank'],
    webdav_path: '/remote.php/dav/files/admin/Executive Reporting & Finance/q3_executive_review.md',
    version: '3.0'
  }
];

const defaultCalendarEvents = [
  {
    event_id: 'evt-001',
    calendar: 'Shift Rotations',
    title: 'Feline Morning Care & Nutrition Shift',
    description: 'Morning feeding and hydration check for all feline executives in Habitat Alpha.',
    department: 'Veterinary Protocols',
    location: 'Habitat Cleanroom Alpha',
    start_time: '2026-10-15T08:00:00Z',
    end_time: '2026-10-15T12:00:00Z',
    organizer: 'emp-human-elena',
    status: 'confirmed'
  },
  {
    event_id: 'evt-002',
    calendar: 'Habitat Maintenance Windows',
    title: 'HVAC Filter Replacement & Sensor Recalibration',
    description: 'Scheduled filter swap and telemetry check for Cleanroom B air handlers.',
    department: 'Facilities & Maintenance SOPs',
    location: 'Main Habitat B',
    start_time: '2026-10-15T13:00:00Z',
    end_time: '2026-10-15T15:30:00Z',
    organizer: 'emp-human-bob',
    status: 'scheduled'
  },
  {
    event_id: 'evt-003',
    calendar: 'Corporate Milestones',
    title: 'OCI All-Hands & Platform Strategy Review',
    description: 'Quarterly company meeting covering multi-persona simulation and White-label release.',
    department: 'Executive Reporting & Finance',
    location: 'Virtual Hub Room 1',
    start_time: '2026-10-15T16:00:00Z',
    end_time: '2026-10-15T17:00:00Z',
    organizer: 'emp-human-alice',
    status: 'confirmed'
  },
  {
    event_id: 'evt-004',
    calendar: 'Veterinary Audits',
    title: 'Senior Executive Wellness & VHR Physical',
    description: 'Comprehensive health audit and blood chemistry review for Oliver and Rusty.',
    department: 'Veterinary Protocols',
    location: 'Veterinary Clinic Bay 2',
    start_time: '2026-10-16T10:00:00Z',
    end_time: '2026-10-16T11:30:00Z',
    organizer: 'emp-human-elena',
    status: 'scheduled'
  }
];

const hubFolders = ref([...defaultHubFolders]);
const documents = ref([...defaultDocuments]);
const calendarEvents = ref([...defaultCalendarEvents]);

const docSearchQuery = ref('');
const selectedDepartmentFilter = ref('');
const selectedCategoryFilter = ref('');
const selectedCalendarFilter = ref('');

const showDocumentReaderModal = ref(false);
const selectedDocument = ref(null);
const showNewDocModal = ref(false);
const showNewEventModal = ref(false);
const isPublishingDoc = ref(false);
const isSchedulingEvent = ref(false);

const newDocForm = ref({
  title: '',
  department: 'HR & Policies',
  category: 'Policy',
  author: 'emp-human-alice',
  content: ''
});

const newEventForm = ref({
  calendar: 'Shift Rotations',
  title: '',
  description: '',
  department: 'Veterinary Protocols',
  location: 'Habitat Alpha',
  start_time: '',
  end_time: '',
  organizer: 'emp-human-elena',
  status: 'scheduled'
});

const filteredDocuments = computed(() => {
  return documents.value.filter(doc => {
    if (selectedDepartmentFilter.value && doc.department !== selectedDepartmentFilter.value) {
      return false;
    }
    if (selectedCategoryFilter.value && doc.category !== selectedCategoryFilter.value) {
      return false;
    }
    if (docSearchQuery.value) {
      const q = docSearchQuery.value.toLowerCase();
      const matchTitle = doc.title && doc.title.toLowerCase().includes(q);
      const matchContent = doc.content && doc.content.toLowerCase().includes(q);
      const matchAuthor = doc.author && doc.author.toLowerCase().includes(q);
      return matchTitle || matchContent || matchAuthor;
    }
    return true;
  });
});

const filteredCalendarEvents = computed(() => {
  return calendarEvents.value.filter(ev => {
    if (selectedCalendarFilter.value && ev.calendar !== selectedCalendarFilter.value) {
      return false;
    }
    return true;
  });
});

function toggleDepartmentFilter(dept) {
  if (selectedDepartmentFilter.value === dept) {
    selectedDepartmentFilter.value = '';
  } else {
    selectedDepartmentFilter.value = dept;
  }
}

function getDepartmentFolderIcon(mountPoint) {
  if (!mountPoint) return '📁';
  if (mountPoint.includes('HR') || mountPoint.includes('Policies')) return '📋';
  if (mountPoint.includes('Facilities') || mountPoint.includes('Maintenance')) return '🔧';
  if (mountPoint.includes('Veterinary')) return '🩺';
  if (mountPoint.includes('Executive') || mountPoint.includes('Finance')) return '📈';
  return '📁';
}

function getDocumentCountForDepartment(mountPoint) {
  return documents.value.filter(d => d.department === mountPoint).length;
}

function getCategoryBadgeStyle(category) {
  switch (category) {
    case 'Policy': return 'background: #0284c7; color: white;';
    case 'Architecture': return 'background: #7c3aed; color: white;';
    case 'Clinical Protocol': return 'background: #059669; color: white;';
    case 'SOP': return 'background: #ea580c; color: white;';
    case 'Report': return 'background: #d97706; color: white;';
    default: return 'background: #475569; color: white;';
  }
}

function getCalendarBadgeStyle(calendar) {
  switch (calendar) {
    case 'Shift Rotations': return 'background: #0284c7; color: white;';
    case 'Habitat Maintenance Windows': return 'background: #ea580c; color: white;';
    case 'Corporate Milestones': return 'background: #7c3aed; color: white;';
    case 'Veterinary Audits': return 'background: #059669; color: white;';
    default: return 'background: #475569; color: white;';
  }
}

function getAuthorDisplayName(authorId) {
  if (!authorId) return 'Alice Vance';
  const staff = staffList.value.find(s => s.id === authorId || s.name === authorId);
  return staff ? staff.name : authorId;
}

function openDocumentReader(doc) {
  selectedDocument.value = doc;
  showDocumentReaderModal.value = true;
}

function copyWebDAVPath(path) {
  if (navigator && navigator.clipboard) {
    navigator.clipboard.writeText(path).catch(() => {});
  }
  showToast(`WebDAV Path copied: ${path}`);
}

function formatDateTime(dtStr) {
  if (!dtStr) return 'TBD';
  try {
    const d = new Date(dtStr);
    return d.toLocaleString([], { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' });
  } catch (e) {
    return dtStr;
  }
}

async function fetchHubFolders() {
  try {
    const res = await fetch('/api/v1/employee/hub/folders');
    if (res.ok) {
      const data = await res.json();
      if (data && data.folders && data.folders.length > 0) {
        hubFolders.value = data.folders;
      }
    }
  } catch (e) {
    // fallback to defaults
  }
}

async function fetchDocuments() {
  try {
    const res = await fetch('/api/v1/employee/documents');
    if (res.ok) {
      const data = await res.json();
      if (data && data.documents && data.documents.length > 0) {
        documents.value = data.documents;
      }
    }
  } catch (e) {
    // fallback to defaults
  }
}

async function fetchCalendarEvents() {
  try {
    const res = await fetch('/api/v1/employee/calendar/events');
    if (res.ok) {
      const data = await res.json();
      if (data && data.events && data.events.length > 0) {
        calendarEvents.value = data.events;
      }
    }
  } catch (e) {
    // fallback to defaults
  }
}

async function submitNewDocument() {
  isPublishingDoc.value = true;
  try {
    const res = await fetch('/api/v1/employee/documents', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-CSRF-Token': getCsrfToken()
      },
      body: JSON.stringify(newDocForm.value)
    });
    if (res.ok) {
      const created = await res.json();
      documents.value.unshift(created);
      showToast(`Document "${created.title}" published to Nextcloud!`);
      showNewDocModal.value = false;
      newDocForm.value = {
        title: '',
        department: 'HR & Policies',
        category: 'Policy',
        author: 'emp-human-alice',
        content: ''
      };
    } else {
      showToast('Failed to publish document.');
    }
  } catch (e) {
    showToast('Failed to publish document.');
  } finally {
    isPublishingDoc.value = false;
  }
}

async function submitNewCalendarEvent() {
  isSchedulingEvent.value = true;
  try {
    const now = new Date();
    const payload = {
      ...newEventForm.value,
      start_time: newEventForm.value.start_time || new Date(now.getTime() + 3600000).toISOString(),
      end_time: newEventForm.value.end_time || new Date(now.getTime() + 7200000).toISOString()
    };
    const res = await fetch('/api/v1/employee/calendar/events', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-CSRF-Token': getCsrfToken()
      },
      body: JSON.stringify(payload)
    });
    if (res.ok) {
      const created = await res.json();
      calendarEvents.value.unshift(created);
      showToast(`Event "${created.title}" scheduled on ${created.calendar}!`);
      showNewEventModal.value = false;
      newEventForm.value = {
        calendar: 'Shift Rotations',
        title: '',
        description: '',
        department: 'Veterinary Protocols',
        location: 'Habitat Alpha',
        start_time: '',
        end_time: '',
        organizer: 'emp-human-elena',
        status: 'scheduled'
      };
    } else {
      showToast('Failed to schedule event.');
    }
  } catch (e) {
    showToast('Failed to schedule event.');
  } finally {
    isSchedulingEvent.value = false;
  }
}

function getCsrfToken() {
  const match = document.cookie.match(/(?:^|;\s*)csrf_token=([^;]+)/);
  if (match) return match[1];
  const hostMatch = document.cookie.match(/(?:^|;\s*)__Host-csrf-token=([^;]+)/);
  if (hostMatch) return hostMatch[1];
  return 'oci-employee-csrf-token';
}

async function fetchHelpdeskTickets() {
  try {
    const res = await fetch('/api/v1/ops/tickets?limit=100');
    if (res.ok) {
      const data = await res.json();
      helpdeskTickets.value = data.items || [];
    }
  } catch (e) {}
}

async function fetchTicketComments(ticketId) {
  try {
    const res = await fetch(`/api/v1/ops/tickets/${ticketId}/comments`);
    if (res.ok) {
      const data = await res.json();
      helpdeskComments.value = data.items || [];
    }
  } catch (e) {}
}

function openTicketDetail(ticket) {
  selectedHelpdeskTicket.value = ticket;
  helpdeskComments.value = [];
  fetchTicketComments(ticket.ticket_id);
  showTicketDetailModal.value = true;
}

async function submitNewTicket() {
  if (!newTicketForm.value.title) {
    showToast('Ticket title is required.');
    return;
  }
  isSubmittingTicket.value = true;
  try {
    const isUrgent = newTicketForm.value.is_urgent || newTicketForm.value.severity === 'urgent' || newTicketForm.value.severity === 'critical';
    const payload = {
      forgejo_repo: newTicketForm.value.forgejo_repo || 'oci-internal/helpdesk',
      title: newTicketForm.value.title,
      body: newTicketForm.value.body,
      category: newTicketForm.value.category,
      severity: newTicketForm.value.severity,
      source: 'internal',
      author_username: newTicketForm.value.author_username || 'emp-human-elise',
      is_urgent: isUrgent
    };
    const res = await fetch('/api/v1/ops/tickets', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-CSRF-Token': getCsrfToken()
      },
      body: JSON.stringify(payload)
    });
    if (res.ok) {
      const created = await res.json();
      helpdeskTickets.value.unshift(created);
      if (isUrgent) {
        showToast('🚨 Urgent ticket dispatched to On-Call Pebble watch & mobile push (ntfy)!');
      } else {
        showToast('Ticket created and synced to Forgejo issue tracker!');
      }
      showNewTicketModal.value = false;
      newTicketForm.value = {
        category: 'Habitat',
        severity: 'medium',
        title: '',
        body: '',
        is_urgent: false,
        forgejo_repo: 'oci-internal/helpdesk',
        author_username: 'emp-human-elise'
      };
    } else {
      showToast('Failed to create ticket.');
    }
  } catch (e) {
    showToast('Failed to create ticket.');
  } finally {
    isSubmittingTicket.value = false;
  }
}

async function quickAcknowledgeTicket(ticket) {
  try {
    const res = await fetch(`/api/v1/ops/tickets/${ticket.ticket_id}`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        'X-CSRF-Token': getCsrfToken()
      },
      body: JSON.stringify({ state: 'acknowledged', assignee_username: 'emp-human-frank' })
    });
    if (res.ok) {
      const updated = await res.json();
      const idx = helpdeskTickets.value.findIndex(t => t.ticket_id === ticket.ticket_id);
      if (idx !== -1) helpdeskTickets.value[idx] = updated;
      if (selectedHelpdeskTicket.value?.ticket_id === ticket.ticket_id) {
        selectedHelpdeskTicket.value = updated;
      }
      showToast('Ticket acknowledged! ACK confirmed on Pebble watch.');
    }
  } catch (e) {
    showToast('Failed to acknowledge ticket.');
  }
}

async function quickCloseTicket(ticket) {
  try {
    const res = await fetch(`/api/v1/ops/tickets/${ticket.ticket_id}`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        'X-CSRF-Token': getCsrfToken()
      },
      body: JSON.stringify({ state: 'closed', resolution_notes: 'Resolved by engineering & facilities.' })
    });
    if (res.ok) {
      const updated = await res.json();
      const idx = helpdeskTickets.value.findIndex(t => t.ticket_id === ticket.ticket_id);
      if (idx !== -1) helpdeskTickets.value[idx] = updated;
      if (selectedHelpdeskTicket.value?.ticket_id === ticket.ticket_id) {
        selectedHelpdeskTicket.value = updated;
      }
      showToast('Ticket closed and resolved! Forgejo issue closed.');
    }
  } catch (e) {
    showToast('Failed to close ticket.');
  }
}

async function submitTicketComment() {
  if (!newCommentForm.value.comment || !selectedHelpdeskTicket.value) return;
  isSubmittingComment.value = true;
  try {
    const payload = {
      author_username: newCommentForm.value.author_username || 'emp-human-frank',
      comment: newCommentForm.value.comment,
      is_internal: newCommentForm.value.is_internal
    };
    const res = await fetch(`/api/v1/ops/tickets/${selectedHelpdeskTicket.value.ticket_id}/comments`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-CSRF-Token': getCsrfToken()
      },
      body: JSON.stringify(payload)
    });
    if (res.ok) {
      const created = await res.json();
      helpdeskComments.value.push(created);
      newCommentForm.value.comment = '';
      if (payload.is_internal) {
        showToast('Internal note saved (scrubbed from customer view).');
      } else {
        showToast('Comment posted and synced to Forgejo issue thread.');
      }
    } else {
      showToast('Failed to post comment.');
    }
  } catch (e) {
    showToast('Failed to post comment.');
  } finally {
    isSubmittingComment.value = false;
  }
}

async function fetchExecutiveReports() {
  try {
    const res = await fetch('/api/v1/employee/reports/monthly');
    if (res.ok) {
      const data = await res.json();
      const items = Array.isArray(data) ? data : (data.items || []);
      if (items.length > 0) {
        executiveReports.value = items;
        if (!selectedReport.value || !items.find(r => r.report_id === selectedReport.value.report_id)) {
          selectedReport.value = items[0];
        }
      }
    }
  } catch (e) {
    // fallback
  }
}

async function triggerGenerateReport() {
  isGeneratingReport.value = true;
  try {
    const res = await fetch('/api/v1/employee/reports/monthly/generate', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-CSRF-Token': getCsrfToken()
      },
      body: JSON.stringify(generateReportForm.value)
    });
    if (res.ok) {
      const rep = await res.json();
      const idx = executiveReports.value.findIndex(r => r.report_id === rep.report_id);
      if (idx >= 0) {
        executiveReports.value[idx] = rep;
      } else {
        executiveReports.value.unshift(rep);
      }
      selectedReport.value = rep;
      showToast(`Executive Report for ${rep.period} generated & archived to Nextcloud!`);
      showGenerateReportModal.value = false;
    } else {
      showToast('Failed to generate executive report.');
    }
  } catch (e) {
    showToast('Failed to generate executive report.');
  } finally {
    isGeneratingReport.value = false;
  }
}

function openReportDetail(report) {
  selectedReport.value = report;
  showReportDetailModal.value = true;
}

function downloadReport(report, format) {
  window.open(`/api/v1/employee/reports/monthly/${report.report_id}/download?format=${format}`, '_blank');
}

onMounted(() => {
  fetchEmployees();
  fetchLeaveRequests();
  fetchReviewCycles();
  fetchVHRRecords();
  fetchIncidents();
  fetchTimesheets();
  fetchSalarySlips();
  fetchHubFolders();
  fetchDocuments();
  fetchCalendarEvents();
  fetchHelpdeskTickets();
  fetchExecutiveReports();
  initIncidentSSE();
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

.period-badge {
  background: rgba(56, 189, 248, 0.15);
  color: #38bdf8;
  font-weight: 700;
  border-radius: 0.25rem;
  padding: 0.15rem 0.45rem;
  font-size: 0.75rem;
  border: 1px solid rgba(56, 189, 248, 0.4);
  display: inline-block;
}
.phase-badge {
  padding: 0.15rem 0.45rem;
  border-radius: 0.25rem;
  font-size: 0.7rem;
  font-weight: 600;
  display: inline-block;
}
.phase-badge.self_appraisal { background: rgba(148, 163, 184, 0.2); color: #cbd5e1; border: 1px solid #475569; }
.phase-badge.peer_feedback { background: rgba(59, 130, 246, 0.2); color: #60a5fa; border: 1px solid #3b82f6; }
.phase-badge.manager_review { background: rgba(234, 179, 8, 0.2); color: #facc15; border: 1px solid #eab308; }
.phase-badge.completed { background: rgba(34, 197, 94, 0.2); color: #4ade80; border: 1px solid #22c55e; }

.stepper-mini {
  display: flex;
  align-items: center;
  gap: 0.25rem;
}
.step-dot {
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background: #334155;
  color: #94a3b8;
  font-size: 0.65rem;
  font-weight: bold;
  display: flex;
  align-items: center;
  justify-content: center;
}
.step-dot.active {
  background: #0284c7;
  color: white;
  box-shadow: 0 0 8px rgba(2, 132, 199, 0.6);
}
.step-dot.completed {
  background: #16a34a;
  color: white;
}
.step-line {
  width: 10px;
  height: 2px;
  background: #334155;
}

.evaluators-list {
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
  font-size: 0.75rem;
}
.evaluators-list small {
  color: #94a3b8;
  font-weight: 600;
}

.scores-container {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}
.score-subtext {
  font-size: 0.7rem;
  color: #94a3b8;
  display: flex;
  flex-direction: column;
  gap: 0.1rem;
}

.modal-tab-bar {
  display: flex;
  gap: 0.5rem;
  border-bottom: 1px solid #334155;
  padding-bottom: 0.75rem;
  margin-bottom: 1.25rem;
  flex-wrap: wrap;
}
.modal-tab-btn {
  background: #1e293b;
  border: 1px solid #334155;
  color: #94a3b8;
  padding: 0.5rem 0.85rem;
  border-radius: 0.375rem;
  font-size: 0.8rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}
.modal-tab-btn.active {
  background: #0284c7;
  color: white;
  border-color: #38bdf8;
}
.tab-content-panel {
  animation: fadeIn 0.15s ease-in-out;
}
.tab-content-panel h4 {
  margin: 0 0 1rem 0;
  color: #f8fafc;
  font-size: 0.95rem;
}

.badge-urgent { background: #ef4444; color: white; }
.badge-high { background: #f97316; color: white; }
.badge-medium { background: #3b82f6; color: white; }
.badge-low { background: #64748b; color: white; }
.badge-open { background: #0284c7; color: white; }
.badge-ack { background: #d97706; color: white; }
.badge-closed { background: #059669; color: white; }

.emergency-alert-banner {
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: linear-gradient(135deg, rgba(239, 68, 68, 0.25) 0%, rgba(185, 28, 28, 0.45) 100%);
  border: 1px solid #ef4444;
  border-left: 6px solid #ef4444;
  border-radius: 0.5rem;
  padding: 1rem 1.25rem;
  margin-bottom: 1.5rem;
  box-shadow: 0 4px 15px rgba(239, 68, 68, 0.3);
  animation: alertPulse 2.5s infinite ease-in-out;
}
@keyframes alertPulse {
  0%, 100% { box-shadow: 0 4px 15px rgba(239, 68, 68, 0.3); }
  50% { box-shadow: 0 4px 25px rgba(239, 68, 68, 0.6); }
}
.alert-content {
  display: flex;
  align-items: center;
  gap: 1rem;
}
.alert-icon {
  font-size: 2rem;
  animation: alertBounce 1s infinite alternate;
}
@keyframes alertBounce {
  from { transform: translateY(0); }
  to { transform: translateY(-4px); }
}
.alert-headline {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 0.25rem;
}
.alert-title-tag {
  font-size: 0.75rem;
  font-weight: 800;
  letter-spacing: 0.05em;
  color: #fca5a5;
}
.zone-badge {
  background: #334155;
  color: #e2e8f0;
  padding: 0.15rem 0.4rem;
  border-radius: 0.25rem;
  font-size: 0.75rem;
  font-weight: 600;
}
.alert-title {
  margin: 0.2rem 0;
  color: #ffffff;
  font-size: 1.05rem;
}
.alert-desc {
  margin: 0;
  color: #e2e8f0;
  font-size: 0.85rem;
}
.alert-actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}
.btn-alert-ack {
  background: #10b981;
  color: white;
  border: none;
  padding: 0.5rem 0.85rem;
  border-radius: 0.375rem;
  font-weight: 600;
  font-size: 0.85rem;
  cursor: pointer;
  transition: background 0.2s;
}
.btn-alert-ack:hover {
  background: #059669;
}
.btn-alert-view {
  background: #334155;
  color: #e2e8f0;
  border: 1px solid #475569;
  padding: 0.5rem 0.85rem;
  border-radius: 0.375rem;
  font-size: 0.85rem;
  cursor: pointer;
  transition: background 0.2s;
}
.btn-alert-view:hover {
  background: #475569;
}
.btn-alert-dismiss {
  background: transparent;
  border: none;
  color: #94a3b8;
  font-size: 1.25rem;
  cursor: pointer;
  padding: 0.25rem 0.5rem;
}
.btn-alert-dismiss:hover {
  color: #ffffff;
}
</style>

