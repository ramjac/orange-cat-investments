<template>
  <div class="container">
    <header>
      <div class="brand">
        <span class="brand-icon">🐱</span>
        <div class="brand-title">
          <h1>Orange Cat Investments</h1>
          <p>Customer Investor Portal — Portfolio, Live Habitat Stream & Momentum Trading</p>
        </div>
      </div>
      <div class="user-profile">
        <div class="avatar">{{ currentPersona === 'arthur' ? 'AP' : 'CS' }}</div>
        <div class="user-info">
          <h4>{{ portfolio.customer_name || (currentPersona === 'arthur' ? 'Arthur Pendelton' : 'Chloe Spark') }}</h4>
          <span>{{ portfolio.account_name || (currentPersona === 'arthur' ? 'Long-Term Value Investor' : 'Short-Term Momentum Alpha Trader') }}</span>
        </div>
        <button class="switch-persona-btn" @click="togglePersona">
          Switch Persona
        </button>
      </div>
    </header>

    <!-- Navigation Tabs -->
    <nav class="nav-tabs">
      <button :class="['tab-btn', { active: currentTab === 'portfolio' }]" @click="currentTab = 'portfolio'">
        📈 Portfolio & Performance
      </button>
      <button :class="['tab-btn', { active: currentTab === 'stream' }]" @click="currentTab = 'stream'">
        📹 Live Habitat Stream (WebRTC)
      </button>
      <button :class="['tab-btn', { active: currentTab === 'trading' }]" @click="currentTab = 'trading'">
        ⚡ Active Trading & Zoomie Index
      </button>
      <button :class="['tab-btn', { active: currentTab === 'deposits' }]" @click="currentTab = 'deposits'">
        🏦 Recurring Deposits (ACH)
      </button>
      <button :class="['tab-btn', { active: currentTab === 'developer' }]" @click="currentTab = 'developer'">
        🔑 API Keys & Webhooks
      </button>
      <button :class="['tab-btn', { active: currentTab === 'esg' }]" @click="currentTab = 'esg'">
        🌿 Feline Welfare & ESG Dashboard
      </button>
    </nav>

    <!-- TAB 1: Portfolio & Performance -->
    <div v-if="currentTab === 'portfolio'" class="grid-layout">
      <div class="card">
        <h3>Portfolio Summary</h3>
        <p class="subtext">Account: {{ portfolio.account_name }} ({{ portfolio.portfolio_id.substring(0, 8) }}...)</p>

        <div class="metrics-grid">
          <div class="metric-box">
            <span class="metric-label">Total Portfolio Value</span>
            <span class="metric-value">${{ portfolio.total_balance_usd.toLocaleString('en-US', {minimumFractionDigits: 2}) }}</span>
          </div>
          <div class="metric-box">
            <span class="metric-label">Cash Balance</span>
            <span class="metric-value">${{ portfolio.cash_balance_usd.toLocaleString('en-US', {minimumFractionDigits: 2}) }}</span>
          </div>
          <div class="metric-box">
            <span class="metric-label">Total Return (YTD)</span>
            <span class="metric-value positive-text">+28.45%</span>
          </div>
        </div>

        <h4>Asset Allocation</h4>
        <div class="table-container">
          <table class="data-table">
            <thead>
              <tr>
                <th>Symbol</th>
                <th>Quantity</th>
                <th>Avg Price</th>
                <th>Current Price</th>
                <th>Market Value</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="holding in portfolio.holdings" :key="holding.symbol">
                <td><strong>{{ holding.symbol }}</strong></td>
                <td>{{ holding.quantity }}</td>
                <td>${{ holding.avg_price.toFixed(2) }}</td>
                <td>${{ holding.current_price.toFixed(2) }}</td>
                <td><strong>${{ holding.market_value.toLocaleString('en-US', {minimumFractionDigits: 2}) }}</strong></td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <div class="card">
        <h3>Historical Performance</h3>
        <p class="subtext">Portfolio valuation growth driven by observation alpha.</p>

        <div class="chart-mock">
          <div v-for="(p, idx) in portfolio.historical_performance" :key="idx" class="chart-bar-container">
            <div class="chart-bar" :style="{ height: (p.valuation / 1500) + 'px' }"></div>
            <span class="chart-label">{{ p.date }}</span>
          </div>
        </div>

        <h4>Recent Automated Execution Logs</h4>
        <div class="table-container">
          <table class="data-table">
            <thead>
              <tr>
                <th>Order ID</th>
                <th>Symbol</th>
                <th>Side</th>
                <th>Qty</th>
                <th>Price</th>
                <th>Trigger Activity</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="trade in tradeLogs" :key="trade.order_id">
                <td><code class="code-font">{{ trade.order_id }}</code></td>
                <td><strong>{{ trade.symbol }}</strong></td>
                <td><span :class="['side-badge', trade.side]">{{ trade.side.toUpperCase() }}</span></td>
                <td>{{ trade.quantity }}</td>
                <td>${{ trade.price }}</td>
                <td><span class="activity-chip">{{ trade.trigger_activity || 'zooming' }}</span></td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- TAB 2: Live Habitat WebRTC Video Stream -->
    <div v-else-if="currentTab === 'stream'" class="grid-layout stream-layout">
      <div class="card">
        <div class="card-header">
          <h2>📹 Live Habitat WebRTC Stream (Alpha Sunbeam Lounge)</h2>
          <span class="live-indicator">🔴 LIVE | Sub-500ms Latency</span>
        </div>

        <div class="video-player-container">
          <div class="video-placeholder">
            <div class="video-overlay">
              <span class="cam-icon">🎥</span>
              <p><strong>RTSP/WebRTC Stream Active:</strong> https://stream.oci.local/webrtc/alpha-lounge-cam1</p>
              <div class="detection-box">
                🎯 <strong>AI Vision Overlay:</strong> Feline Executive (98.2% Confidence — Activity: <em>Zooming</em>)
              </div>
            </div>
          </div>
        </div>

        <div class="stream-controls">
          <button class="btn-secondary" @click="showToast('Camera angle adjusted to Alpha Perch Zone 1')">🔄 Recenter Camera</button>
          <button class="btn-secondary" @click="showToast('Audio stream unmuted (Purr frequency: 28Hz)')">🔊 Unmute Habitat Mic</button>
          <span class="stream-stat">Stream Quality: 4K 60fps</span>
        </div>
      </div>

      <div class="card">
        <h3>Camera Stream Catalog</h3>
        <p class="subtext">Select active optical sensor stream</p>

        <div class="stream-list">
          <div v-for="stream in streams" :key="stream.stream_id" class="stream-card">
            <h4>{{ stream.name }}</h4>
            <small>ID: {{ stream.stream_id }}</small>
            <div class="stream-meta">
              <span>Protocol: <strong>{{ stream.protocol.toUpperCase() }}</strong></span>
              <span class="status-badge completed">{{ stream.status }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- TAB 3: Active Trading Terminal & Zoomie Index -->
    <div v-else-if="currentTab === 'trading'" class="grid-layout trading-layout">
      <div class="card">
        <div class="card-header">
          <h2>⚡ Live "Zoomie Index" & Momentum Alerts</h2>
          <span class="zoomie-badge">SCORE: {{ zoomieIndex.zoomie_index_score }} / 100</span>
        </div>

        <div class="zoomie-banner">
          <div class="zoomie-info">
            <h3>🚀 {{ zoomieIndex.momentum_level }}</h3>
            <p>Leader: <strong>{{ zoomieIndex.feline_leader }}</strong> | Signal: <strong class="positive-text">{{ zoomieIndex.trading_signal }}</strong> ({{ zoomieIndex.recommended_multiplier }}x Multiplier)</p>
          </div>
          <button class="btn-primary-glow" @click="executeQuickTrade">
            🔥 1-Click Zoomie Pounce Execution
          </button>
        </div>

        <h3>Active Ticker Stream</h3>
        <div class="ticker-grid">
          <div v-for="item in tickerData" :key="item.symbol" class="ticker-card">
            <span class="ticker-symbol">{{ item.symbol }}</span>
            <span class="ticker-price">${{ item.price.toFixed(2) }}</span>
            <span :class="['ticker-change', item.change_pct >= 0 ? 'pos' : 'neg']">
              {{ item.change_pct >= 0 ? '+' : '' }}{{ item.change_pct }}%
            </span>
          </div>
        </div>
      </div>

      <div class="card">
        <h3>1-Click Order Execution</h3>
        <form @submit.prevent="submitOrder">
          <div class="form-group">
            <label>Symbol</label>
            <select class="form-control" v-model="orderForm.symbol" required>
              <option value="AAPL">AAPL — Apple Inc.</option>
              <option value="NVDA">NVDA — NVIDIA Corp.</option>
              <option value="MSFT">MSFT — Microsoft Corp.</option>
              <option value="TSLA">TSLA — Tesla Inc.</option>
            </select>
          </div>

          <div class="form-group">
            <label>Action</label>
            <select class="form-control" v-model="orderForm.side" required>
              <option value="buy">BUY</option>
              <option value="sell">SELL</option>
            </select>
          </div>

          <div class="form-group">
            <label>Quantity</label>
            <input type="number" class="form-control" v-model.number="orderForm.quantity" min="1" required />
          </div>

          <div class="form-group">
            <label>Order Type</label>
            <select class="form-control" v-model="orderForm.order_type">
              <option value="market">Market Order</option>
              <option value="bracket">Bracket Order (Stop-Loss + Take-Profit)</option>
              <option value="trailing_stop">Trailing Stop-Loss</option>
            </select>
          </div>

          <button type="submit" class="btn-primary" :disabled="isSubmittingOrder">
            {{ isSubmittingOrder ? '⌛ Routing Order...' : '🚀 Submit Order' }}
          </button>
        </form>
      </div>
    </div>

    <!-- TAB 4: Recurring Deposits (ACH) -->
    <div v-else-if="currentTab === 'deposits'" class="grid-layout deposit-layout">
      <div class="card">
        <h3>Automated Recurring Deposit Scheduler</h3>
        <p class="subtext">Schedule passive automated dollar-cost averaging into OCI investment funds.</p>

        <form @submit.prevent="submitDeposit">
          <div class="form-group">
            <label>Amount (USD)</label>
            <input type="number" class="form-control" v-model.number="depositForm.amount_usd" step="50" min="50" required />
          </div>

          <div class="form-group">
            <label>Frequency</label>
            <select class="form-control" v-model="depositForm.frequency" required>
              <option value="monthly">Monthly</option>
              <option value="biweekly">Bi-Weekly</option>
              <option value="weekly">Weekly</option>
            </select>
          </div>

          <div class="form-group">
            <label>Linked Bank Account (ACH)</label>
            <input type="text" class="form-control" v-model="depositForm.bank_account" placeholder="e.g. Chase Checking (****4821)" required />
          </div>

          <button type="submit" class="btn-primary" :disabled="isSubmittingDeposit">
            {{ isSubmittingDeposit ? '⌛ Scheduling...' : '💳 Enable Recurring ACH Deposit' }}
          </button>
        </form>
      </div>

      <div class="card">
        <h3>Active Deposit Schedules</h3>
        <div class="table-container">
          <table class="data-table">
            <thead>
              <tr>
                <th>Deposit ID</th>
                <th>Amount</th>
                <th>Frequency</th>
                <th>Bank Account</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="dep in deposits" :key="dep.deposit_id">
                <td><code class="code-font">{{ dep.deposit_id }}</code></td>
                <td><strong>${{ dep.amount_usd.toFixed(2) }}</strong></td>
                <td>{{ dep.frequency }}</td>
                <td>{{ dep.bank_account }}</td>
                <td><span class="status-badge completed">{{ dep.status.toUpperCase() }}</span></td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- TAB 5: Developer API Keys & Webhooks -->
    <div v-else-if="currentTab === 'developer'" class="grid-layout dev-layout">
      <div class="card">
        <h3>Generate Developer API Key</h3>
        <p class="subtext">Connect custom quant trading bots via HMAC Bearer tokens.</p>

        <form @submit.prevent="createAPIKey">
          <div class="form-group">
            <label>Key Description</label>
            <input type="text" class="form-control" v-model="apiKeyForm.name" placeholder="e.g. Chloe Momentum Quant Bot" required />
          </div>

          <button type="submit" class="btn-primary">🔑 Generate API Key</button>
        </form>

        <div v-if="newlyCreatedKey" class="key-box">
          <p>⚠️ <strong>Copy your secret key now!</strong> It won't be shown again.</p>
          <code class="code-font key-text">{{ newlyCreatedKey }}</code>
        </div>
      </div>

      <div class="card">
        <h3>Webhook Subscriptions</h3>
        <form @submit.prevent="createWebhook">
          <div class="form-group">
            <label>Target Webhook Endpoint URL</label>
            <input type="url" class="form-control" v-model="webhookForm.target_url" placeholder="https://your-bot.com/webhook" required />
          </div>
          <button type="submit" class="btn-primary">🔔 Subscribe Webhook</button>
        </form>

        <h4>Active Subscriptions</h4>
        <div class="table-container">
          <table class="data-table">
            <thead>
              <tr>
                <th>Sub ID</th>
                <th>Target URL</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="wh in webhooks" :key="wh.subscription_id">
                <td><code class="code-font">{{ wh.subscription_id }}</code></td>
                <td><span class="url-text">{{ wh.target_url }}</span></td>
                <td><span class="status-badge completed">{{ wh.status.toUpperCase() }}</span></td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- TAB 6: ESG & Feline Welfare Dashboard -->
    <div v-if="currentTab === 'esg'" class="grid-layout esg-layout">
      <div class="card">
        <div class="card-header">
          <h2>🌿 Feline Welfare & ESG Transparency Dashboard</h2>
          <span class="welfare-badge">Score: {{ esgData.overall_welfare_score.toFixed(2) }} / 5.0</span>
        </div>

        <div class="metrics-grid">
          <div class="metric-box">
            <span class="metric-label">Veterinary Adherence</span>
            <span class="metric-value positive-text">{{ esgData.veterinary_adherence_pct.toFixed(1) }}%</span>
          </div>
          <div class="metric-box">
            <span class="metric-label">Workplace Incidents</span>
            <span class="metric-value">{{ esgData.workplace_incidents_cnt }}</span>
          </div>
          <div class="metric-box">
            <span class="metric-label">Perch Comfort Score</span>
            <span class="metric-value">{{ esgData.perch_comfort_rating }}</span>
          </div>
        </div>

        <h3>Feline Executive Health & Wellbeing Index</h3>
        <div class="feline-esg-grid">
          <div v-for="exec in esgData.feline_executives" :key="exec.name" class="feline-esg-card">
            <h4>🐱 {{ exec.name }} — {{ exec.role }}</h4>
            <p>Health Status: <strong class="positive-text">{{ exec.health_status }}</strong></p>
            <p>Whisker Symmetry: <strong>{{ exec.whisker_symmetry }}</strong></p>
            <p>Purr Frequency: <strong>{{ exec.purr_frequency_hz }} Hz</strong></p>
            <p>Sunbeam Zone: <strong>{{ exec.preferred_sunbeam }}</strong></p>
            <p v-if="exec.vhr_records_cnt !== undefined">VHR Checkups Recorded: <strong>{{ exec.vhr_records_cnt }}</strong></p>
          </div>
        </div>
      </div>
    </div>

    <div v-if="toastMessage" class="alert-toast">
      {{ toastMessage }}
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue';

const currentPersona = ref('arthur');
const currentTab = ref('portfolio');
const toastMessage = ref('');
const isSubmittingOrder = ref(false);
const isSubmittingDeposit = ref(false);
const newlyCreatedKey = ref('');

const portfolio = ref({
  portfolio_id: '018f3a9a-2222-7000-8000-000000000002',
  account_name: 'OCI Alpha Growth Fund',
  total_balance_usd: 128450.75,
  cash_balance_usd: 32100.25,
  holdings: [
    { symbol: 'AAPL', quantity: 120.0, avg_price: 178.50, current_price: 224.30, market_value: 26916.00 },
    { symbol: 'NVDA', quantity: 85.0, avg_price: 110.20, current_price: 138.80, market_value: 11798.00 },
    { symbol: 'MSFT', quantity: 90.0, avg_price: 380.00, current_price: 448.20, market_value: 40338.00 },
    { symbol: 'TSLA', quantity: 70.0, avg_price: 190.00, current_price: 247.12, market_value: 17298.40 }
  ],
  historical_performance: [
    { date: 'Jan 2026', valuation: 100000.00 },
    { date: 'Feb 2026', valuation: 108400.00 },
    { date: 'Mar 2026', valuation: 114200.00 },
    { date: 'Apr 2026', valuation: 128450.75 }
  ]
});

const tradeLogs = ref([
  { order_id: 'ord-98234-a1', symbol: 'AAPL', side: 'buy', quantity: 15.0, price: 224.30, trigger_activity: 'zooming' },
  { order_id: 'ord-98233-a2', symbol: 'NVDA', side: 'buy', quantity: 10.0, price: 138.80, trigger_activity: 'playful_pounce' }
]);

const streams = ref([
  { stream_id: 'str-018f-0001', name: 'Alpha Sunbeam Lounge Cam 1', protocol: 'webrtc', status: 'active' },
  { stream_id: 'str-018f-0002', name: '4K Feline Perch Scope', protocol: 'webrtc', status: 'active' }
]);

const zoomieIndex = ref({
  zoomie_index_score: 88.5,
  momentum_level: 'HIGH_VELOCITY_3AM_ZOOMIES',
  feline_leader: 'Feline Executive',
  trading_signal: 'TACTICAL_BUY_BULLISH',
  recommended_multiplier: 1.75
});

const tickerData = ref([
  { symbol: 'AAPL', price: 224.30, change_pct: 1.45 },
  { symbol: 'NVDA', price: 138.80, change_pct: 3.12 },
  { symbol: 'MSFT', price: 448.20, change_pct: 0.82 },
  { symbol: 'TSLA', price: 247.12, change_pct: -0.65 }
]);

const deposits = ref([
  { deposit_id: 'dep-018f-01', amount_usd: 500.00, frequency: 'monthly', bank_account: 'Chase Checking (****4821)', status: 'active' }
]);

const webhooks = ref([
  { subscription_id: 'sub-018f-1111', target_url: 'https://quant-bot.chloespark.io/api/v1/oci-callback', status: 'active' }
]);

const esgData = ref({
  overall_welfare_score: 5.0,
  veterinary_adherence_pct: 100.0,
  workplace_incidents_cnt: 0,
  perch_comfort_rating: '5.0 / 5.0 (Ergonomic Thermal Cushioning)',
  feline_executives: []
});

const orderForm = ref({ symbol: 'AAPL', side: 'buy', quantity: 10, order_type: 'market' });
const depositForm = ref({ amount_usd: 500, frequency: 'monthly', bank_account: 'Chase Checking (****4821)' });
const apiKeyForm = ref({ name: '' });
const webhookForm = ref({ target_url: '' });

async function fetchESGTransparency() {
  try {
    const res = await fetch('/api/v1/customer/transparency/welfare');
    if (res.ok) {
      const data = await res.json();
      if (data && data.overall_welfare_score !== undefined) {
        esgData.value = data;
      }
    }
  } catch (e) {
    // Local fallback
  }
}

async function fetchPortfolio() {
  try {
    const res = await fetch(`/api/v1/customer/portfolio?customer_id=${currentPersona.value}`);
    if (res.ok) {
      const data = await res.json();
      if (data && data.portfolio_id) {
        portfolio.value = data;
      }
    }
  } catch (e) {
    // Fallback
  }
}

function togglePersona() {
  if (currentPersona.value === 'arthur') {
    currentPersona.value = 'chloe';
    currentTab.value = 'trading';
    fetchPortfolio();
    showToast('Switched to Chloe Spark (Momentum Trader mode)');
  } else {
    currentPersona.value = 'arthur';
    currentTab.value = 'portfolio';
    fetchPortfolio();
    showToast('Switched to Arthur Pendelton (Long-Term Investor mode)');
  }
}

function showToast(msg) {
  toastMessage.value = msg;
  setTimeout(() => { toastMessage.value = ''; }, 3500);
}

async function executeQuickTrade() {
  showToast('⚡ Executing 1-Click Zoomie Pounce Order for 25 NVDA shares...');
  tradeLogs.value.unshift({
    order_id: 'ord-' + Math.random().toString(36).substring(2, 8),
    symbol: 'NVDA',
    side: 'buy',
    quantity: 25.0,
    price: 138.80,
    trigger_activity: '3AM_ZOOMIES'
  });
}

async function submitOrder() {
  isSubmittingOrder.value = true;
  setTimeout(() => {
    tradeLogs.value.unshift({
      order_id: 'ord-' + Math.random().toString(36).substring(2, 8),
      symbol: orderForm.value.symbol,
      side: orderForm.value.side,
      quantity: orderForm.value.quantity,
      price: 224.30,
      trigger_activity: 'manual_terminal'
    });
    isSubmittingOrder.value = false;
    showToast(`Order executed: ${orderForm.value.side.toUpperCase()} ${orderForm.value.quantity} ${orderForm.value.symbol}`);
  }, 600);
}

async function submitDeposit() {
  isSubmittingDeposit.value = true;
  setTimeout(() => {
    deposits.value.unshift({
      deposit_id: 'dep-' + Math.random().toString(36).substring(2, 8),
      amount_usd: depositForm.value.amount_usd,
      frequency: depositForm.value.frequency,
      bank_account: depositForm.value.bank_account,
      status: 'active'
    });
    isSubmittingDeposit.value = false;
    showToast(`Recurring ACH deposit scheduled for $${depositForm.value.amount_usd}`);
  }, 600);
}

async function createAPIKey() {
  newlyCreatedKey.value = 'oci_live_' + Math.random().toString(36).substring(2) + Math.random().toString(36).substring(2);
  showToast(`API Key generated for ${apiKeyForm.value.name}`);
}

async function createWebhook() {
  webhooks.value.unshift({
    subscription_id: 'sub-' + Math.random().toString(36).substring(2, 8),
    target_url: webhookForm.value.target_url,
    status: 'active'
  });
  showToast(`Webhook subscribed to ${webhookForm.value.target_url}`);
}

onMounted(() => {
  fetchPortfolio();
  fetchESGTransparency();
});
</script>

<style scoped>
.container { max-width: 1100px; margin: 0 auto; color: var(--text-color, #f8fafc); font-family: var(--font-family, sans-serif); }
header { display: flex; justify-content: space-between; align-items: center; border-bottom: 1px solid #334155; padding-bottom: 1rem; margin-bottom: 1.5rem; }
.brand { display: flex; align-items: center; gap: 0.75rem; }
.brand-title h1 { margin: 0; color: var(--secondary-color, #ea580c); font-size: 1.5rem; }
.brand-title p { margin: 0; color: #94a3b8; font-size: 0.85rem; }
.user-profile { display: flex; align-items: center; gap: 0.75rem; background: var(--card-color, #1e293b); padding: 0.5rem 1rem; border-radius: 9999px; }
.avatar { width: 36px; height: 36px; border-radius: 50%; background: var(--accent-color, #3b82f6); display: flex; align-items: center; justify-content: center; font-weight: bold; }
.user-info h4 { margin: 0; font-size: 0.85rem; }
.user-info span { font-size: 0.75rem; color: #94a3b8; }
.switch-persona-btn { background: #334155; border: none; color: var(--text-color, #f8fafc); padding: 0.35rem 0.75rem; border-radius: 0.25rem; cursor: pointer; font-size: 0.75rem; }

.nav-tabs { display: flex; gap: 0.5rem; margin-bottom: 1.5rem; flex-wrap: wrap; }
.tab-btn { background: var(--card-color, #1e293b); border: 1px solid #334155; color: #94a3b8; padding: 0.6rem 1rem; border-radius: 0.5rem; font-weight: bold; cursor: pointer; transition: all 0.2s; font-size: 0.85rem; }
.tab-btn.active { background: var(--secondary-color, #ea580c); color: white; border-color: var(--secondary-color, #ea580c); }

.grid-layout { display: grid; grid-template-columns: 1fr 1fr; gap: 1.5rem; }
.card { background: var(--card-color, #1e293b); border-radius: 0.75rem; border: 1px solid #334155; padding: 1.5rem; }
.subtext { font-size: 0.85rem; color: #94a3b8; margin-top: 0.25rem; margin-bottom: 1rem; }

.metrics-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 1rem; margin-bottom: 1.5rem; }
.metric-box { background: var(--background-color, #0f172a); padding: 1rem; border-radius: 0.5rem; border: 1px solid #334155; }
.metric-label { display: block; font-size: 0.75rem; color: #94a3b8; }
.metric-value { font-size: 1.25rem; font-weight: bold; }
.positive-text { color: #4ade80; }

.table-container { overflow-x: auto; margin-top: 1rem; }
.data-table { width: 100%; border-collapse: collapse; text-align: left; font-size: 0.85rem; }
.data-table th { padding: 0.6rem; border-bottom: 1px solid #334155; color: #94a3b8; }
.data-table td { padding: 0.6rem; border-bottom: 1px solid #1e293b; }

.chart-mock { display: flex; align-items: flex-end; gap: 1.5rem; height: 120px; border-bottom: 1px solid #334155; padding-bottom: 0.5rem; margin-bottom: 1.5rem; }
.chart-bar-container { display: flex; flex-direction: column; align-items: center; flex: 1; }
.chart-bar { width: 100%; background: var(--secondary-color, #ea580c); border-radius: 0.25rem 0.25rem 0 0; }
.chart-label { font-size: 0.75rem; color: #94a3b8; margin-top: 0.35rem; }

.side-badge { padding: 0.15rem 0.4rem; border-radius: 0.2rem; font-size: 0.7rem; font-weight: bold; }
.side-badge.buy { background: rgba(34, 197, 94, 0.2); color: #4ade80; }
.side-badge.sell { background: rgba(239, 68, 68, 0.2); color: #f87171; }
.activity-chip { background: #334155; padding: 0.15rem 0.4rem; border-radius: 9999px; font-size: 0.7rem; color: #e2e8f0; }

.video-player-container { background: var(--background-color, #0f172a); border-radius: 0.5rem; height: 220px; display: flex; align-items: center; justify-content: center; position: relative; border: 1px solid #334155; }
.video-overlay { text-align: center; }
.cam-icon { font-size: 2.5rem; display: block; }
.detection-box { background: rgba(234, 88, 12, 0.15); border: 1px solid var(--secondary-color, #ea580c); color: #fdba74; padding: 0.5rem 1rem; border-radius: 0.375rem; margin-top: 0.75rem; font-size: 0.85rem; }
.card-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem; }
.live-indicator { color: #ef4444; font-weight: bold; font-size: 0.8rem; }
.stream-controls { display: flex; gap: 0.75rem; align-items: center; margin-top: 1rem; }
.btn-secondary { background: #334155; color: white; border: none; padding: 0.5rem 1rem; border-radius: 0.375rem; font-size: 0.8rem; cursor: pointer; }
.stream-stat { font-size: 0.75rem; color: #94a3b8; margin-left: auto; }

.zoomie-badge { background: var(--secondary-color, #ea580c); color: white; font-weight: bold; padding: 0.2rem 0.6rem; border-radius: 0.25rem; font-size: 0.8rem; }
.zoomie-banner { background: rgba(234, 88, 12, 0.15); border: 1px solid var(--secondary-color, #ea580c); border-radius: 0.5rem; padding: 1rem; display: flex; justify-content: space-between; align-items: center; margin-bottom: 1.5rem; }
.zoomie-info h3 { margin: 0 0 0.25rem 0; color: #fdba74; font-size: 1.1rem; }
.zoomie-info p { margin: 0; font-size: 0.85rem; color: #cbd5e1; }
.btn-primary-glow { background: var(--secondary-color, #ea580c); color: white; border: none; padding: 0.75rem 1.25rem; border-radius: 0.375rem; font-weight: bold; cursor: pointer; box-shadow: 0 0 15px rgba(234, 88, 12, 0.5); }

.ticker-grid { display: grid; grid-template-columns: repeat(2, 1fr); gap: 0.75rem; margin-top: 1rem; }
.ticker-card { background: var(--background-color, #0f172a); padding: 0.75rem; border-radius: 0.375rem; border: 1px solid #334155; display: flex; justify-content: space-between; align-items: center; }
.ticker-symbol { font-weight: bold; }
.ticker-price { font-weight: 600; }
.ticker-change.pos { color: #4ade80; }
.ticker-change.neg { color: #f87171; }

.form-group { margin-bottom: 1rem; }
.form-group label { display: block; font-size: 0.8rem; color: #94a3b8; margin-bottom: 0.35rem; }
.form-control { width: 100%; background: var(--background-color, #0f172a); border: 1px solid #334155; color: var(--text-color, #f8fafc); padding: 0.6rem; border-radius: 0.375rem; box-sizing: border-box; }
.btn-primary { background: var(--secondary-color, #ea580c); color: white; border: none; padding: 0.75rem; border-radius: 0.375rem; font-weight: bold; cursor: pointer; width: 100%; }

.key-box { background: var(--background-color, #0f172a); border: 1px solid var(--accent-color, #3b82f6); padding: 1rem; border-radius: 0.5rem; margin-top: 1rem; }
.key-text { font-size: 0.9rem; color: #60a5fa; word-break: break-all; }
.code-font { font-family: monospace; }
.alert-toast { position: fixed; bottom: 2rem; right: 2rem; background: var(--secondary-color, #ea580c); color: white; padding: 1rem 1.5rem; border-radius: 0.5rem; box-shadow: 0 10px 15px -3px rgba(0,0,0,0.5); font-weight: bold; }
.welfare-badge { background: #16a34a; color: white; padding: 0.2rem 0.6rem; border-radius: 0.25rem; font-size: 0.8rem; font-weight: bold; }
.feline-esg-grid { display: grid; grid-template-columns: repeat(2, 1fr); gap: 1rem; margin-top: 1rem; }
.feline-esg-card { background: var(--background-color, #0f172a); padding: 1rem; border-radius: 0.5rem; border: 1px solid #334155; }
.feline-esg-card h4 { margin: 0 0 0.5rem 0; color: var(--secondary-color, #ea580c); }
.feline-esg-card p { margin: 0.25rem 0; font-size: 0.85rem; color: #cbd5e1; }
</style>
