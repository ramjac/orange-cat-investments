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
        <div class="avatar">{{ currentTab === 'care' ? 'ER' : 'DQ' }}</div>
        <div class="user-info">
          <h4>{{ currentTab === 'care' ? 'Dr. Elena Rostova' : 'David Quant' }}</h4>
          <span>{{ currentTab === 'care' ? 'Chief Veterinary Officer' : 'Feline Behavioral Data Scientist' }}</span>
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

    <div v-if="toastMessage" class="alert-toast">
      {{ toastMessage }}
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue';

const currentTab = ref('care');

const felineRoster = ref([
  { id: 'emp-feline-garfield', name: 'Garfield', title: 'Chief Observation Officer', avatar: '🐱' },
  { id: 'emp-feline-barneby', name: 'Barneby', title: 'Senior Alpha Perch Analyst', avatar: '🐈‍⬛' }
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

function showToast(msg) {
  toastMessage.value = msg;
  setTimeout(() => { toastMessage.value = ''; }, 3500);
}

function selectFeline(id) {
  selectedFelineId.value = id;
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
.positive-text { color: #4ade80; font-weight: bold; }
</style>
