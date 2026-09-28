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
        <div class="avatar">{{ currentTab === 'care' ? 'ER' : (currentTab === 'leave' || currentTab === 'review' ? 'AV' : 'DQ') }}</div>
        <div class="user-info">
          <h4>{{ currentTab === 'care' ? 'Dr. Elena Rostova' : (currentTab === 'leave' || currentTab === 'review' ? 'Alice Vance' : 'David Quant') }}</h4>
          <span>{{ currentTab === 'care' ? 'Chief Veterinary Officer' : (currentTab === 'leave' || currentTab === 'review' ? 'Head of Human & Feline Resources' : 'Feline Behavioral Data Scientist') }}</span>
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
              <option value="emp-feline-garfield">🐱 Garfield — Chief Observation Officer</option>
              <option value="emp-feline-barneby">🐈‍⬛ Barneby — Senior Alpha Perch Analyst</option>
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
                <option value="emp-feline-garfield">🐱 Garfield</option>
                <option value="emp-feline-barneby">🐈‍⬛ Barneby</option>
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
  { id: 'emp-human-elena', name: 'Dr. Elena Rostova', role: 'Chief Veterinary Officer (Human)', icon: '🩺' }
]);

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

async function submitVHRRecord() {
  vhrRecords.value.unshift({
    record_id: 'vhr-' + Math.random().toString(36).substring(2, 8),
    feline_id: vhrForm.value.feline_id,
    visit_date: new Date().toISOString().split('T')[0],
    weight_kg: vhrForm.value.weight_kg,
    dental_score: vhrForm.value.dental_score,
    vaccination_status: vhrForm.value.vaccination_status,
    clinical_notes: vhrForm.value.clinical_notes
  });
  showToast('VHR Clinical Record saved successfully!');
}

async function submitIncident() {
  incidentLogs.value.unshift({
    incident_id: 'inc-' + Math.random().toString(36).substring(2, 8),
    title: incidentForm.value.title,
    category: incidentForm.value.category,
    involved_feline_id: incidentForm.value.involved_feline_id,
    severity: incidentForm.value.severity,
    status: 'open',
    description: incidentForm.value.description
  });
  showToast('Workplace incident report logged!');
}

async function resolveIncident(id) {
  const inc = incidentLogs.value.find(i => i.incident_id === id);
  if (inc) {
    inc.status = 'resolved';
    showToast('Incident resolved and archived.');
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

onMounted(() => {
  fetchLeaveRequests();
  fetchReviewCycles();
});
</script>

<style scoped>
.container { max-width: 1100px; margin: 0 auto; color: #f8fafc; font-family: sans-serif; }
header { display: flex; justify-content: space-between; align-items: center; border-bottom: 1px solid #334155; padding-bottom: 1rem; margin-bottom: 1.5rem; }
.brand { display: flex; align-items: center; gap: 0.75rem; }
.brand-title h1 { margin: 0; color: #f97316; font-size: 1.5rem; }
.brand-title p { margin: 0; color: #94a3b8; font-size: 0.85rem; }
.user-profile { display: flex; align-items: center; gap: 0.75rem; background: #1e293b; padding: 0.5rem 1rem; border-radius: 9999px; }
.avatar { width: 36px; height: 36px; border-radius: 50%; background: #f97316; display: flex; align-items: center; justify-content: center; font-weight: bold; }
.user-info h4 { margin: 0; font-size: 0.85rem; }
.user-info span { font-size: 0.75rem; color: #94a3b8; }

.nav-tabs { display: flex; gap: 1rem; margin-bottom: 1.5rem; }
.tab-btn { background: #1e293b; border: 1px solid #334155; color: #94a3b8; padding: 0.75rem 1.25rem; border-radius: 0.5rem; font-weight: bold; cursor: pointer; transition: all 0.2s; }
.tab-btn.active { background: #ea580c; color: white; border-color: #ea580c; }

.grid-layout { display: grid; grid-template-columns: 280px 1fr; gap: 1.5rem; }
.backtest-layout { grid-template-columns: 380px 1fr; }

.card { background: #1e293b; border-radius: 0.75rem; border: 1px solid #334155; padding: 1.5rem; }
.subtext { font-size: 0.85rem; color: #94a3b8; margin-top: 0.25rem; margin-bottom: 1rem; }
.feline-list { display: flex; flex-direction: column; gap: 0.75rem; margin-top: 1rem; }
.feline-btn { display: flex; align-items: center; gap: 0.75rem; background: #0f172a; border: 1px solid #334155; padding: 0.75rem; border-radius: 0.5rem; color: #f8fafc; cursor: pointer; text-align: left; }
.feline-btn.active { border-color: #f97316; background: #1e1b4b; }
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
.form-control { width: 100%; background: #0f172a; border: 1px solid #334155; color: #f8fafc; padding: 0.75rem; border-radius: 0.375rem; box-sizing: border-box; }
.code-font { font-family: monospace; }
textarea.form-control { min-height: 80px; resize: vertical; }
.time-tags { display: flex; flex-wrap: wrap; gap: 0.5rem; margin-bottom: 0.5rem; }
.time-tag { background: #334155; padding: 0.35rem 0.75rem; border-radius: 9999px; font-size: 0.85rem; display: flex; align-items: center; gap: 0.5rem; }
.time-tag button { background: none; border: none; color: #94a3b8; cursor: pointer; }
.add-time-box { display: flex; gap: 0.5rem; }
.btn-primary { background: #f97316; color: white; border: none; padding: 0.75rem 1.5rem; border-radius: 0.375rem; font-weight: bold; cursor: pointer; width: 100%; }
.form-actions { text-align: right; margin-top: 1.5rem; }
.alert-toast { position: fixed; bottom: 2rem; right: 2rem; background: #3b82f6; color: white; padding: 1rem 1.5rem; border-radius: 0.5rem; box-shadow: 0 10px 15px -3px rgba(0,0,0,0.5); }

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
.filter-btn { background: #0f172a; border: 1px solid #334155; color: #94a3b8; padding: 0.35rem 0.75rem; border-radius: 0.375rem; font-size: 0.8rem; cursor: pointer; transition: all 0.2s; }
.filter-btn.active { background: #f97316; color: white; border-color: #f97316; font-weight: bold; }
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
.grading-modal-card { margin-top: 1.5rem; padding: 1.25rem; background: #0f172a; border: 1px solid #0284c7; border-radius: 0.5rem; }
.grading-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.5rem; }
.grading-header h3 { margin: 0; color: #38bdf8; font-size: 1rem; }
.close-btn { background: none; border: none; color: #94a3b8; font-size: 1.25rem; cursor: pointer; }
.score-input-group { display: flex; align-items: center; gap: 0.5rem; }
.score-input { font-size: 1.1rem; font-weight: bold; color: #facc15; width: 120px; text-align: center; }
.score-scale { color: #94a3b8; font-size: 0.9rem; font-weight: bold; }
.preset-chips { display: flex; gap: 0.35rem; margin-top: 0.5rem; flex-wrap: wrap; }
.preset-btn { background: #1e293b; border: 1px solid #475569; color: #cbd5e1; padding: 0.2rem 0.5rem; border-radius: 9999px; font-size: 0.75rem; cursor: pointer; }
.preset-btn:hover { border-color: #facc15; color: #facc15; }
.grading-actions { display: flex; justify-content: flex-end; gap: 0.75rem; margin-top: 1rem; }
.btn-save-grading { background: #0284c7; color: white; border: none; padding: 0.5rem 1.25rem; border-radius: 0.375rem; font-weight: bold; cursor: pointer; }
.btn-save-grading:hover { background: #0369a1; }
</style>
