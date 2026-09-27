<template>
  <div class="container">
    <header>
      <div class="brand">
        <span class="brand-icon">🐈</span>
        <div class="brand-title">
          <h1>Orange Cat Investments</h1>
          <p>Employee Portal — Feline Care & Trading Hold Management</p>
        </div>
      </div>
      <div class="user-profile">
        <div class="avatar">ER</div>
        <div class="user-info">
          <h4>Dr. Elena Rostova</h4>
          <span>Chief Veterinary Officer</span>
        </div>
      </div>
    </header>

    <div class="grid-layout">
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

    <div v-if="toastMessage" class="alert-toast">
      {{ toastMessage }}
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue';

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
</script>

<style scoped>
.container { max-width: 1100px; margin: 0 auto; color: #f8fafc; font-family: sans-serif; }
header { display: flex; justify-content: space-between; align-items: center; border-bottom: 1px solid #334155; padding-bottom: 1rem; margin-bottom: 2rem; }
.brand { display: flex; align-items: center; gap: 0.75rem; }
.brand-title h1 { margin: 0; color: #f97316; font-size: 1.5rem; }
.brand-title p { margin: 0; color: #94a3b8; font-size: 0.85rem; }
.user-profile { display: flex; align-items: center; gap: 0.75rem; background: #1e293b; padding: 0.5rem 1rem; border-radius: 9999px; }
.avatar { width: 36px; height: 36px; border-radius: 50%; background: #f97316; display: flex; align-items: center; justify-content: center; font-weight: bold; }
.user-info h4 { margin: 0; font-size: 0.85rem; }
.user-info span { font-size: 0.75rem; color: #94a3b8; }
.grid-layout { display: grid; grid-template-columns: 280px 1fr; gap: 1.5rem; }
.card { background: #1e293b; border-radius: 0.75rem; border: 1px solid #334155; padding: 1.5rem; }
.subtext { font-size: 0.85rem; color: #94a3b8; margin-top: 0.25rem; }
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
.form-group label { display: block; font-size: 0.85rem; color: #94a3b8; margin-bottom: 0.5rem; }
.form-control { width: 100%; background: #0f172a; border: 1px solid #334155; color: #f8fafc; padding: 0.75rem; border-radius: 0.375rem; box-sizing: border-box; }
textarea.form-control { min-height: 80px; resize: vertical; }
.time-tags { display: flex; flex-wrap: wrap; gap: 0.5rem; margin-bottom: 0.5rem; }
.time-tag { background: #334155; padding: 0.35rem 0.75rem; border-radius: 9999px; font-size: 0.85rem; display: flex; align-items: center; gap: 0.5rem; }
.time-tag button { background: none; border: none; color: #94a3b8; cursor: pointer; }
.add-time-box { display: flex; gap: 0.5rem; }
.btn-primary { background: #f97316; color: white; border: none; padding: 0.75rem 1.5rem; border-radius: 0.375rem; font-weight: bold; cursor: pointer; }
.form-actions { text-align: right; margin-top: 1.5rem; }
.alert-toast { position: fixed; bottom: 2rem; right: 2rem; background: #3b82f6; color: white; padding: 1rem 1.5rem; border-radius: 0.5rem; box-shadow: 0 10px 15px -3px rgba(0,0,0,0.5); }
</style>
