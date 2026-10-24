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
      <div class="search-bar-container">
        <span class="search-icon">🔍</span>
        <input
          v-model="catalogSearchQuery"
          type="text"
          placeholder="Search funds, tickers, ESG scores..."
          @input="performCatalogSearch"
          @focus="showSearchDropdown = true"
        />
        <div v-if="showSearchDropdown && catalogSearchResults.length > 0" class="search-dropdown-menu">
          <div
            v-for="res in catalogSearchResults"
            :key="res.id"
            class="search-dropdown-item"
            @click="selectCatalogResult(res)"
          >
            <span class="item-tag" :class="res.category">{{ (res.category || 'ITEM').toUpperCase() }}</span>
            <div class="item-text">
              <strong>{{ res.title }}</strong>
              <small>{{ res.description }}</small>
            </div>
          </div>
        </div>
      </div>
      <div class="user-profile">
        <div class="avatar">{{ currentPersona === 'arthur' ? 'AP' : 'CS' }}</div>
        <div class="user-info">
          <h4>{{ portfolio.customer_name || (currentPersona === 'arthur' ? 'Arthur Pendelton' : 'Chloe Spark') }}</h4>
          <span>{{ portfolio.account_name || (currentPersona === 'arthur' ? 'Long-Term Value Investor' : 'Short-Term Momentum Alpha Trader') }}</span>
        </div>
        <a href="https://status.oci.local" target="_blank" class="switch-persona-btn" style="text-decoration: none; color: #4ade80; display: inline-flex; align-items: center; gap: 4px;" title="Real-Time System Uptime & Status">
          🟢 Status
        </a>
        <button class="switch-persona-btn" @click="togglePersona" style="margin-right: 0.5rem;">
          Switch Persona
        </button>
        <button class="switch-persona-btn" style="background: var(--secondary-color, #ea580c); color: white;" @click="showRegisterModal = true">
          ✨ Register New Account
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
      <button :class="['tab-btn', { active: currentTab === 'support' }]" @click="currentTab = 'support'">
        🎧 Support & Inquiries
      </button>
      <button :class="['tab-btn', { active: currentTab === 'statements' }]" @click="currentTab = 'statements'">
        📄 Statements & Tax Forms
      </button>
      <button :class="['tab-btn', { active: currentTab === 'privacy' }]" @click="currentTab = 'privacy'">
        🔒 Account & GDPR Privacy
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
          <h2>📹 Live Habitat WebRTC Stream ({{ selectedStream?.name || 'Alpha Sunbeam Lounge' }})</h2>
          <span class="live-indicator">🔴 LIVE | Sub-500ms Latency</span>
        </div>

        <div class="video-player-container">
          <div class="video-wrapper">
            <video
              ref="videoPlayer"
              class="video-element"
              autoplay
              playsinline
              :muted="isAudioMuted"
            ></video>
            <div class="video-overlay" v-if="streamStatus !== 'error'">
              <span class="cam-icon">🎥</span>
              <p><strong>{{ (selectedStream?.protocol || 'WebRTC').toUpperCase() }} Stream Active:</strong> {{ selectedStream?.stream_url || 'https://stream.oci.local/webrtc/alpha-lounge-cam1' }}</p>
              <div class="detection-box">
                🎯 <strong>AI Vision Overlay:</strong> Feline Executive (98.2% Confidence — Activity: <em>Zooming</em>)
              </div>
            </div>
            <div v-else class="stream-error-overlay">
              <p>⚠️ Stream connection unavailable. Attempting auto-reconnect...</p>
              <button class="btn-primary" @click="initWebRTCStream()">Retry Stream</button>
            </div>
          </div>
        </div>

        <div class="stream-controls">
          <button class="btn-secondary" @click="recenterCamera()">🔄 Recenter Camera</button>
          <button class="btn-secondary" @click="toggleAudio()">{{ isAudioMuted ? '🔊 Unmute Habitat Mic' : '🔇 Mute Habitat Mic' }}</button>
          <button class="btn-secondary" @click="initWebRTCStream()">⚡ Reconnect WebRTC</button>
          <span class="stream-stat">Status: <strong>{{ streamStatus.toUpperCase() }}</strong> (4K 60fps)</span>
        </div>
      </div>

      <div class="card">
        <h3>Camera Stream Catalog</h3>
        <p class="subtext">Select active optical sensor stream</p>

        <div class="stream-list">
          <div
            v-for="stream in streams"
            :key="stream.stream_id"
            class="stream-card"
            :class="{ active: stream.stream_id === selectedStreamId }"
            @click="selectStream(stream)"
            style="cursor: pointer;"
          >
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

    <!-- TAB: Support & Inquiries -->
    <div v-if="currentTab === 'support'" class="grid-layout" style="grid-template-columns: 1fr 1.2fr; gap: 1.5rem;">
      <!-- Left Column: Submit Ticket Form & Ticket History -->
      <div style="display: flex; flex-direction: column; gap: 1.5rem;">
        <!-- Submit Ticket Card -->
        <div class="card">
          <h3>🎧 Customer Support & Bug Intake</h3>
          <p class="subtext">
            Submit transaction inquiries or report UI issues. Synced with our dedicated engineering support tracker (<code>oci-internal/customer-support</code>).
          </p>

          <form @submit.prevent="submitCustomerTicket" style="display: flex; flex-direction: column; gap: 1rem; margin-top: 1rem;">
            <div>
              <label class="form-label" style="font-size: 0.8rem; color: #94a3b8; display: block; margin-bottom: 0.35rem;">Inquiry Category *</label>
              <select v-model="ticketForm.category" class="score-input" style="width: 100%; padding: 0.6rem; font-size: 0.85rem; background: #0f172a; border: 1px solid #334155; color: #f8fafc; border-radius: 0.375rem;">
                <option value="Trading">⚡ Order Execution & Trading Desk</option>
                <option value="Banking">🏦 Banking & ACH Capital Deposits</option>
                <option value="Habitat">🐾 Feline Habitat Stream & Video</option>
                <option value="UI Bug">🐛 Portal Interface Bug / Visual Issue</option>
                <option value="General">💬 General Account & Policy Inquiry</option>
              </select>
            </div>

            <div>
              <label class="form-label" style="font-size: 0.8rem; color: #94a3b8; display: block; margin-bottom: 0.35rem;">Subject / Title *</label>
              <input
                v-model="ticketForm.title"
                required
                class="score-input"
                style="width: 100%; padding: 0.6rem; font-size: 0.85rem; text-align: left; background: #0f172a; border: 1px solid #334155; color: #f8fafc; border-radius: 0.375rem;"
                placeholder="e.g. Delayed Fill Confirmation for AAPL Order"
              />
            </div>

            <div>
              <label class="form-label" style="font-size: 0.8rem; color: #94a3b8; display: block; margin-bottom: 0.35rem;">Description & Details *</label>
              <textarea
                v-model="ticketForm.body"
                required
                rows="4"
                class="score-input"
                style="width: 100%; padding: 0.6rem; font-size: 0.85rem; text-align: left; resize: vertical; background: #0f172a; border: 1px solid #334155; color: #f8fafc; border-radius: 0.375rem;"
                placeholder="Please describe what occurred, including dates, approximate trade amounts, or browser behavior..."
              ></textarea>
            </div>

            <button type="submit" class="submit-btn" :disabled="isSubmittingSupportTicket" style="padding: 0.75rem; font-weight: bold; cursor: pointer;">
              {{ isSubmittingSupportTicket ? 'Submitting Ticket...' : '📩 Submit Support Ticket' }}
            </button>
          </form>
        </div>

        <!-- Ticket History Card -->
        <div class="card">
          <h3>Your Inquiries</h3>
          <p class="subtext">Select an inquiry to view updates and reply to our support desk.</p>

          <div style="display: flex; flex-direction: column; gap: 0.75rem; margin-top: 1rem;">
            <div
              v-for="t in customerTickets"
              :key="t.ticket_id"
              :class="['stream-card', { active: selectedTicket && selectedTicket.ticket_id === t.ticket_id }]"
              style="padding: 0.85rem; cursor: pointer; border-radius: 0.5rem;"
              @click="selectCustomerTicket(t)"
            >
              <div style="display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 0.35rem;">
                <strong style="color: #f8fafc; font-size: 0.9rem;">{{ t.title }}</strong>
                <span :class="['badge', getTicketStatusBadgeClass(t.state)]">
                  {{ t.state ? t.state.toUpperCase() : 'OPEN' }}
                </span>
              </div>
              <div style="display: flex; justify-content: space-between; font-size: 0.75rem; color: #94a3b8;">
                <span>🏷️ {{ t.category || 'General' }}</span>
                <span>{{ formatDate(t.created_at) }}</span>
              </div>
            </div>

            <div v-if="customerTickets.length === 0" style="text-align: center; color: #94a3b8; padding: 1.5rem; font-size: 0.85rem;">
              No support inquiries submitted yet.
            </div>
          </div>
        </div>
      </div>

      <!-- Right Column: Selected Ticket Details & Discussion Thread -->
      <div class="card" style="display: flex; flex-direction: column; height: 100%;">
        <div v-if="selectedTicket" style="display: flex; flex-direction: column; height: 100%;">
          <div style="border-bottom: 1px solid #334155; padding-bottom: 1rem; margin-bottom: 1rem;">
            <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.5rem;">
              <span class="badge" style="background: #0284c7; color: white;">#{{ selectedTicket.forgejo_issue_id || selectedTicket.ticket_id.substring(0, 6) }}</span>
              <span :class="['badge', getTicketStatusBadgeClass(selectedTicket.state)]">{{ selectedTicket.state ? selectedTicket.state.toUpperCase() : 'OPEN' }}</span>
            </div>
            <h3 style="margin: 0 0 0.5rem 0; font-size: 1.2rem; color: #f8fafc;">{{ selectedTicket.title }}</h3>
            <div style="font-size: 0.8rem; color: #94a3b8;">
              Category: <strong style="color: #cbd5e1;">{{ selectedTicket.category }}</strong> • Submitted: {{ formatDate(selectedTicket.created_at) }}
            </div>
            <div v-if="selectedTicket.body" style="margin-top: 0.75rem; font-size: 0.85rem; color: #e2e8f0; background: #0f172a; padding: 0.75rem; border-radius: 0.375rem; white-space: pre-wrap;">
              {{ selectedTicket.body }}
            </div>
            <div v-if="selectedTicket.resolution_notes" style="margin-top: 0.75rem; font-size: 0.85rem; color: #86efac; background: rgba(34, 197, 94, 0.1); border: 1px solid rgba(34, 197, 94, 0.3); padding: 0.75rem; border-radius: 0.375rem;">
              <strong>Resolution:</strong> {{ selectedTicket.resolution_notes }}
            </div>
          </div>

          <!-- Discussion Messages Thread -->
          <h4 style="margin: 0 0 0.75rem 0; font-size: 0.95rem; color: #f8fafc;">Support Desk Updates & Conversation</h4>
          <div style="flex: 1; overflow-y: auto; display: flex; flex-direction: column; gap: 0.75rem; max-height: 400px; padding-right: 0.5rem; margin-bottom: 1rem;">
            <div
              v-for="msg in ticketComments"
              :key="msg.comment_id"
              :style="{
                background: msg.author_username === portfolio.customer_name || msg.author_username === currentPersona ? '#0f172a' : '#1e293b',
                border: msg.author_username === portfolio.customer_name || msg.author_username === currentPersona ? '1px solid #334155' : '1px solid #0284c7',
                borderRadius: '0.5rem',
                padding: '0.75rem'
              }"
            >
              <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.3rem;">
                <strong style="font-size: 0.8rem; color: #38bdf8;">
                  {{ msg.author_username === portfolio.customer_name || msg.author_username === currentPersona ? 'You (' + msg.author_username + ')' : '🛡️ ' + msg.author_username + ' (OCI Support Desk)' }}
                </strong>
                <span style="font-size: 0.7rem; color: #94a3b8;">{{ formatDate(msg.created_at) }}</span>
              </div>
              <div style="font-size: 0.85rem; color: #f1f5f9; white-space: pre-wrap;">
                {{ msg.comment }}
              </div>
            </div>

            <div v-if="ticketComments.length === 0" style="text-align: center; color: #94a3b8; padding: 2rem; font-size: 0.85rem;">
              No responses yet from support. We typically respond within 1 business hour.
            </div>
          </div>

          <!-- Reply Form -->
          <form @submit.prevent="submitTicketComment" style="display: flex; flex-direction: column; gap: 0.5rem; margin-top: auto;">
            <textarea
              v-model="newCommentText"
              rows="2"
              required
              class="score-input"
              style="width: 100%; padding: 0.5rem; font-size: 0.85rem; text-align: left; resize: vertical; background: #0f172a; border: 1px solid #334155; color: #f8fafc; border-radius: 0.375rem;"
              placeholder="Type your response to the support team..."
            ></textarea>
            <div style="display: flex; justify-content: flex-end;">
              <button type="submit" class="submit-btn" :disabled="isSubmittingComment" style="padding: 0.5rem 1rem; font-size: 0.85rem; cursor: pointer;">
                {{ isSubmittingComment ? 'Sending...' : '💬 Send Reply' }}
              </button>
            </div>
          </form>
        </div>

        <div v-else style="display: flex; flex-direction: column; align-items: center; justify-content: center; height: 100%; color: #94a3b8; padding: 3rem 1rem; text-align: center;">
          <div style="font-size: 3rem; margin-bottom: 1rem;">🎧</div>
          <h4 style="margin: 0 0 0.5rem 0; color: #f8fafc;">Select a Support Inquiry</h4>
          <p style="font-size: 0.85rem; max-width: 320px; margin: 0;">
            Choose a ticket from the left panel to review sanitized engineering updates and reply to our support desk.
          </p>
        </div>
      </div>
    </div>

    <!-- TAB: Statements & Tax Forms -->
    <div v-if="currentTab === 'statements'" class="grid-layout">
      <!-- Left Column: Generate Document on-demand -->
      <div class="card">
        <h3>📄 Financial Documents & Tax Reports</h3>
        <p class="subtext">
          Generate IRS Form 1099-B tax returns or monthly portfolio statements rendered directly to pure Go PDF 1.4 format.
        </p>

        <div style="display: flex; flex-direction: column; gap: 1.5rem; margin-top: 1.25rem;">
          <!-- Monthly Statement Generator -->
          <div style="background: var(--background-color, #0f172a); padding: 1.25rem; border-radius: 0.5rem; border: 1px solid #334155;">
            <h4 style="margin: 0 0 0.5rem 0; color: #fdba74;">📑 Generate Monthly Statement</h4>
            <p style="font-size: 0.8rem; color: #94a3b8; margin-bottom: 1rem;">
              Summarizes monthly equity performance, month-end holdings valuation, cash yields, and observation trade execution records.
            </p>
            <div style="display: flex; gap: 0.75rem; margin-bottom: 1rem;">
              <div style="flex: 1;">
                <label class="form-label" style="font-size: 0.75rem; color: #94a3b8; display: block; margin-bottom: 0.25rem;">Month</label>
                <select v-model="statementMonth" class="form-control" style="padding: 0.5rem; font-size: 0.85rem;">
                  <option value="01">01 - January</option>
                  <option value="02">02 - February</option>
                  <option value="03">03 - March</option>
                  <option value="04">04 - April</option>
                  <option value="05">05 - May</option>
                  <option value="06">06 - June</option>
                  <option value="07">07 - July</option>
                  <option value="08">08 - August</option>
                  <option value="09">09 - September</option>
                  <option value="10">10 - October</option>
                  <option value="11">11 - November</option>
                  <option value="12">12 - December</option>
                </select>
              </div>
              <div style="flex: 1;">
                <label class="form-label" style="font-size: 0.75rem; color: #94a3b8; display: block; margin-bottom: 0.25rem;">Year</label>
                <select v-model="statementYear" class="form-control" style="padding: 0.5rem; font-size: 0.85rem;">
                  <option :value="2026">2026</option>
                  <option :value="2025">2025</option>
                </select>
              </div>
            </div>
            <button
              class="btn-primary"
              :disabled="isGeneratingMonthly"
              @click="generateMonthlyStatementWorkflow"
              style="font-size: 0.85rem; padding: 0.6rem;"
            >
              {{ isGeneratingMonthly ? 'Rendering PDF...' : '📑 Generate Monthly Statement PDF' }}
            </button>
          </div>

          <!-- Form 1099-B Tax Return Generator -->
          <div style="background: var(--background-color, #0f172a); padding: 1.25rem; border-radius: 0.5rem; border: 1px solid #334155;">
            <h4 style="margin: 0 0 0.5rem 0; color: #6ee7b7;">📋 Generate IRS Form 1099-B</h4>
            <p style="font-size: 0.8rem; color: #94a3b8; margin-bottom: 1rem;">
              Annual Proceeds From Broker and Barter Exchange Transactions report showing short/long-term capital gain calculations under 26 U.S.C. § 6045.
            </p>
            <div style="margin-bottom: 1rem;">
              <label class="form-label" style="font-size: 0.75rem; color: #94a3b8; display: block; margin-bottom: 0.25rem;">Tax Year</label>
              <select v-model="taxYear" class="form-control" style="padding: 0.5rem; font-size: 0.85rem;">
                <option :value="2025">Tax Year 2025 (Annual Filing)</option>
                <option :value="2024">Tax Year 2024 (Prior Year)</option>
              </select>
            </div>
            <button
              class="btn-primary"
              :disabled="isGeneratingTax"
              @click="generateForm1099BWorkflow"
              style="background: #059669; font-size: 0.85rem; padding: 0.6rem;"
            >
              {{ isGeneratingTax ? 'Compiling 1099-B...' : '📋 Generate Form 1099-B Tax Return PDF' }}
            </button>
          </div>
        </div>
      </div>

      <!-- Right Column: Document History Table & Direct Downloads -->
      <div class="card">
        <div style="display: flex; justify-content: space-between; align-items: center;">
          <h3>Available Documents & Archival</h3>
          <button class="btn-secondary" style="font-size: 0.75rem; padding: 0.35rem 0.65rem;" @click="fetchStatements">
            🔄 Refresh
          </button>
        </div>
        <p class="subtext">Authenticated PDF 1.4 documents ready for secure browser preview or download.</p>

        <div class="table-container">
          <table class="data-table">
            <thead>
              <tr>
                <th>Type</th>
                <th>Period / Year</th>
                <th>Created</th>
                <th>Size</th>
                <th style="text-align: right;">Action</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="stmt in statements" :key="stmt.statement_id">
                <td>
                  <span :class="['badge', stmt.statement_type === '1099_b' ? 'badge-ack' : 'badge-open']">
                    {{ stmt.statement_type === '1099_b' ? 'FORM 1099-B' : 'MONTHLY' }}
                  </span>
                </td>
                <td><strong>{{ stmt.period_label }}</strong></td>
                <td style="color: #94a3b8; font-size: 0.75rem;">{{ formatDate(stmt.created_at) }}</td>
                <td style="color: #94a3b8; font-size: 0.75rem;">{{ stmt.file_size_bytes ? (stmt.file_size_bytes / 1024).toFixed(1) + ' KB' : 'N/A' }}</td>
                <td style="text-align: right;">
                  <button
                    class="btn-secondary"
                    style="font-size: 0.75rem; padding: 0.3rem 0.6rem; background: #0284c7;"
                    @click="downloadStatement(stmt)"
                  >
                    📥 Download PDF
                  </button>
                </td>
              </tr>
              <tr v-if="statements.length === 0">
                <td colspan="5" style="text-align: center; color: #94a3b8; padding: 2rem;">
                  No statements or tax reports generated yet. Click generate on the left to compile your first report.
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- TAB: Account & GDPR Privacy -->
    <div v-if="currentTab === 'privacy'" class="grid-layout">
      <!-- Left Column: Account Details & Liquidation -->
      <div class="card">
        <h3>🔒 Account Lifecycle & Liquidation</h3>
        <p class="subtext">
          Manage your account state or initiate automated portfolio liquidation.
        </p>

        <div style="background: var(--background-color, #0f172a); padding: 1.25rem; border-radius: 0.5rem; border: 1px solid #334155; margin-bottom: 1.5rem;">
          <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.75rem;">
            <span style="font-size: 0.85rem; color: #94a3b8;">Account Status</span>
            <span :class="['badge', portfolio.status === 'closed' ? 'badge-closed' : 'badge-open']">
              {{ portfolio.status ? portfolio.status.toUpperCase() : 'ACTIVE' }}
            </span>
          </div>
          <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.5rem; font-size: 0.85rem;">
            <span style="color: #94a3b8;">Investor Name:</span>
            <strong>{{ portfolio.customer_name || 'Arthur Pendelton' }}</strong>
          </div>
          <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.5rem; font-size: 0.85rem;">
            <span style="color: #94a3b8;">Portfolio Reference:</span>
            <code class="code-font">{{ portfolio.portfolio_id?.substring(0, 18) }}...</code>
          </div>
          <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.5rem; font-size: 0.85rem;">
            <span style="color: #94a3b8;">Active Equity Positions:</span>
            <strong>{{ portfolio.holdings?.length || 0 }} Open Positions</strong>
          </div>
          <div style="display: flex; justify-content: space-between; align-items: center; font-size: 0.85rem;">
            <span style="color: #94a3b8;">Total Portfolio Value:</span>
            <strong class="positive-text">${{ portfolio.total_balance_usd?.toLocaleString('en-US', {minimumFractionDigits: 2}) }}</strong>
          </div>
        </div>

        <div style="border: 1px solid #7f1d1d; background: rgba(239, 68, 68, 0.08); padding: 1.25rem; border-radius: 0.5rem;">
          <h4 style="margin: 0 0 0.5rem 0; color: #f87171;">⚠️ Close Account & Liquidate Portfolio</h4>
          <p style="font-size: 0.8rem; color: #cbd5e1; margin-bottom: 1rem;">
            This workflow executes market sell orders for all active holdings via the broker gateway, converts your equity to cash, transfers all net cash proceeds back to your linked bank account via Plaid ACH disbursement, and marks your portfolio as closed.
          </p>
          <button
            class="btn-primary"
            style="background: #dc2626;"
            :disabled="portfolio.status === 'closed'"
            @click="showCloseModal = true"
          >
            {{ portfolio.status === 'closed' ? 'Account Is Closed' : '⚠️ Liquidate Portfolio & Close Account' }}
          </button>
        </div>

        <!-- Liquidation Receipt Card if closed -->
        <div v-if="closureReceipt" style="margin-top: 1.5rem; background: rgba(34, 197, 94, 0.1); border: 1px solid #16a34a; padding: 1rem; border-radius: 0.5rem;">
          <h4 style="margin: 0 0 0.5rem 0; color: #4ade80;">✅ Account Closure & Liquidation Complete</h4>
          <div style="font-size: 0.8rem; color: #e2e8f0; display: flex; flex-direction: column; gap: 0.35rem;">
            <div>Liquidated Positions: <strong>{{ closureReceipt.liquidated_positions_count }}</strong></div>
            <div>Gross Proceeds: <strong>${{ closureReceipt.gross_proceeds_usd?.toFixed(2) }}</strong></div>
            <div>Net Disbursed Cash: <strong class="positive-text">${{ closureReceipt.disbursed_cash_usd?.toFixed(2) }}</strong></div>
            <div>ACH Disbursement ID: <code class="code-font">{{ closureReceipt.disbursement_transfer_id }}</code></div>
            <div>Destination Account: <strong>{{ closureReceipt.disbursement_account }}</strong></div>
          </div>
        </div>
      </div>

      <!-- Right Column: GDPR Article 17 Erasure -->
      <div class="card">
        <h3>🛡️ GDPR Right to be Forgotten (Article 17)</h3>
        <p class="subtext">
          Permanently scrub customer PII, purge active Valkey sessions, and receive a cryptographically signed deletion certificate.
        </p>

        <div style="background: var(--background-color, #0f172a); padding: 1.25rem; border-radius: 0.5rem; border: 1px solid #334155; margin-bottom: 1.5rem;">
          <h4 style="margin: 0 0 0.5rem 0; color: #38bdf8;">Compliance Framework & Retention Guardrails</h4>
          <p style="font-size: 0.8rem; color: #94a3b8; line-height: 1.4; margin-bottom: 0.75rem;">
            In accordance with GDPR Article 17(1) and SEC Rule 17a-4 / FINRA financial regulatory standards:
          </p>
          <ul style="font-size: 0.78rem; color: #cbd5e1; padding-left: 1.2rem; margin: 0; display: flex; flex-direction: column; gap: 0.35rem;">
            <li><strong>PII Scrubbing:</strong> Names, emails, passwords, and contact tokens are permanently deleted from <code>core_invest.customers</code>.</li>
            <li><strong>Session Invalidation:</strong> All active sessions and refresh cookies are purged from Valkey session store immediately.</li>
            <li><strong>Audit Anonymization:</strong> Order books, trades, and ledgers are pseudonymized with SHA-256 hashes to retain audit integrity.</li>
            <li><strong>Cryptographic Proof:</strong> An HMAC-SHA256 signed deletion certificate is issued as immutable legal proof of erasure.</li>
          </ul>
        </div>

        <button
          class="btn-primary"
          style="background: #991b1b;"
          @click="showGDPRModal = true"
        >
          🗑️ Request GDPR Data Erasure
        </button>

        <!-- Erasure Certificate Display if created -->
        <div v-if="erasureCertificate" style="margin-top: 1.5rem; background: var(--background-color, #0f172a); border: 1px solid #0284c7; padding: 1.25rem; border-radius: 0.5rem;">
          <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.75rem;">
            <h4 style="margin: 0; color: #38bdf8;">📜 Tamper-Evident Deletion Certificate</h4>
            <span class="badge" style="background: #0284c7; color: white;">HMAC-SHA256 VERIFIED</span>
          </div>
          <div style="display: flex; flex-direction: column; gap: 0.4rem; font-size: 0.8rem;">
            <div>Certificate ID: <code class="code-font">{{ erasureCertificate.certificate_id }}</code></div>
            <div>Issued At: <strong>{{ formatDate(erasureCertificate.issued_at) }}</strong></div>
            <div>Scrubbed Entities: <span class="activity-chip">{{ erasureCertificate.entities_affected?.join(', ') }}</span></div>
            <div>Cryptographic Signature:</div>
            <code class="code-font" style="font-size: 0.75rem; color: #6ee7b7; word-break: break-all; background: #020617; padding: 0.4rem; border-radius: 0.25rem;">
              {{ erasureCertificate.hmac_signature }}
            </code>
          </div>
          <div style="margin-top: 1rem; display: flex; justify-content: flex-end;">
            <button
              class="btn-secondary"
              style="background: #0284c7; font-size: 0.8rem; padding: 0.5rem 1rem;"
              @click="downloadGDPRCertificate(erasureCertificate.certificate_id)"
            >
              📜 Download Legal Certificate (PDF 1.4)
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- MODAL 1: Customer Self-Registration & Onboarding -->
    <div v-if="showRegisterModal" class="modal-overlay" @click.self="showRegisterModal = false">
      <div class="modal-card">
        <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem;">
          <h3 style="margin: 0; color: var(--secondary-color, #ea580c);">✨ Open a New Investment Account</h3>
          <button class="btn-secondary" style="padding: 0.25rem 0.5rem;" @click="showRegisterModal = false">✕</button>
        </div>
        <p class="subtext">
          Onboard directly to OCI with automated ZITADEL OIDC identity creation, NIST-compliant password validation, and instant zero-balance portfolio generation.
        </p>

        <form @submit.prevent="handleRegisterCustomer" style="display: flex; flex-direction: column; gap: 0.85rem;">
          <div class="form-group" style="margin: 0;">
            <label>Legal Full Name *</label>
            <input v-model="registerForm.name" required class="form-control" placeholder="e.g. Arthur Pendelton" />
          </div>

          <div class="form-group" style="margin: 0;">
            <label>Email Address *</label>
            <input v-model="registerForm.email" type="email" required class="form-control" placeholder="e.g. arthur@investor.com" />
          </div>

          <div class="form-group" style="margin: 0;">
            <label>Password (NIST 800-63B standard) *</label>
            <input v-model="registerForm.password" type="password" required class="form-control" placeholder="At least 8 chars, mixed case, number, symbol" />
            <div style="display: flex; gap: 0.5rem; margin-top: 0.35rem; align-items: center; font-size: 0.75rem;">
              <span style="color: #94a3b8;">Strength:</span>
              <span :style="{ color: passwordStrengthColor, fontWeight: 'bold' }">{{ passwordStrengthLabel }}</span>
              <div style="flex: 1; height: 6px; background: #334155; border-radius: 3px; overflow: hidden;">
                <div :style="{ width: passwordStrengthPct + '%', background: passwordStrengthColor, height: '100%', transition: 'all 0.3s' }"></div>
              </div>
            </div>
            <div style="font-size: 0.7rem; color: #94a3b8; margin-top: 0.25rem;">
              Must contain 8+ characters, uppercase, lowercase, a digit, and a special character (!@#$%^&*).
            </div>
          </div>

          <div class="form-group" style="margin: 0;">
            <label>Linked Bank Account (ACH via Plaid) *</label>
            <input v-model="registerForm.bank_account" required class="form-control" placeholder="e.g. Chase Premier Checking (****4821)" />
          </div>

          <div class="form-group" style="margin: 0;">
            <label>Initial Cash Deposit (USD)</label>
            <input v-model.number="registerForm.initial_deposit" type="number" step="100" min="0" class="form-control" placeholder="0.00" />
          </div>

          <div style="display: flex; justify-content: flex-end; gap: 0.75rem; margin-top: 0.5rem;">
            <button type="button" class="btn-secondary" @click="showRegisterModal = false">Cancel</button>
            <button type="submit" class="btn-primary" :disabled="isRegistering || !isPasswordValid" style="width: auto; padding: 0.6rem 1.25rem;">
              {{ isRegistering ? 'Creating Account...' : '🚀 Create Account' }}
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- MODAL 2: Account Closure Confirmation -->
    <div v-if="showCloseModal" class="modal-overlay" @click.self="showCloseModal = false">
      <div class="modal-card">
        <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem;">
          <h3 style="margin: 0; color: #f87171;">⚠️ Confirm Account Liquidation</h3>
          <button class="btn-secondary" style="padding: 0.25rem 0.5rem;" @click="showCloseModal = false">✕</button>
        </div>
        <p class="subtext">
          Are you sure you want to close account <strong>{{ portfolio.account_name }}</strong>?
        </p>

        <div style="background: var(--background-color, #0f172a); padding: 1rem; border-radius: 0.5rem; border: 1px solid #7f1d1d; margin-bottom: 1rem; font-size: 0.85rem; color: #fca5a5;">
          <strong>Warning:</strong> All {{ portfolio.holdings?.length || 0 }} open equity positions will be sold immediately at market price. All cash proceeds will be transferred to your linked bank account via Plaid ACH disbursement.
        </div>

        <label style="display: flex; align-items: center; gap: 0.5rem; font-size: 0.85rem; color: #cbd5e1; margin-bottom: 1.25rem; cursor: pointer;">
          <input type="checkbox" v-model="closeConfirmChecked" />
          I confirm that I want to liquidate my entire portfolio and close this account.
        </label>

        <div style="display: flex; justify-content: flex-end; gap: 0.75rem;">
          <button type="button" class="btn-secondary" @click="showCloseModal = false">Cancel</button>
          <button
            type="button"
            class="btn-primary"
            style="background: #dc2626; width: auto; padding: 0.6rem 1.25rem;"
            :disabled="!closeConfirmChecked || isClosingAccount"
            @click="handleCloseAccountWorkflow"
          >
            {{ isClosingAccount ? 'Liquidating Positions...' : 'Confirm & Liquidate' }}
          </button>
        </div>
      </div>
    </div>

    <!-- MODAL 3: GDPR Article 17 Erasure Confirmation -->
    <div v-if="showGDPRModal" class="modal-overlay" @click.self="showGDPRModal = false">
      <div class="modal-card">
        <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem;">
          <h3 style="margin: 0; color: #ef4444;">🛡️ GDPR Article 17 Erasure Request</h3>
          <button class="btn-secondary" style="padding: 0.25rem 0.5rem;" @click="showGDPRModal = false">✕</button>
        </div>
        <p class="subtext">
          Permanent data deletion is irreversible. Your profile, credentials, and active sessions will be completely purged.
        </p>

        <div style="background: var(--background-color, #0f172a); padding: 1rem; border-radius: 0.5rem; border: 1px solid #334155; margin-bottom: 1rem; font-size: 0.85rem;">
          <p style="margin: 0 0 0.5rem 0; color: #94a3b8;">To prevent accidental data loss, please type <strong style="color: #f87171;">DELETE</strong> to confirm:</p>
          <input
            v-model="gdprConfirmText"
            class="form-control"
            placeholder="Type DELETE to confirm"
            style="font-family: monospace; font-size: 0.9rem;"
          />
        </div>

        <div style="display: flex; justify-content: flex-end; gap: 0.75rem;">
          <button type="button" class="btn-secondary" @click="showGDPRModal = false">Cancel</button>
          <button
            type="button"
            class="btn-primary"
            style="background: #991b1b; width: auto; padding: 0.6rem 1.25rem;"
            :disabled="gdprConfirmText !== 'DELETE' || isErasingData"
            @click="handleGDPRErasureWorkflow"
          >
            {{ isErasingData ? 'Erasing Personal Data...' : 'Permanently Delete My Data' }}
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
import { ref, computed, watch, onMounted, onUnmounted } from 'vue';

const currentPersona = ref('arthur');
const currentTab = ref('portfolio');
const toastMessage = ref('');
const isSubmittingOrder = ref(false);
const isSubmittingDeposit = ref(false);
const newlyCreatedKey = ref('');
const csrfToken = ref('');

const catalogSearchQuery = ref('');
const catalogSearchResults = ref([]);
const showSearchDropdown = ref(false);

const performCatalogSearch = async () => {
  if (!catalogSearchQuery.value) {
    catalogSearchResults.value = [];
    return;
  }
  try {
    const res = await fetch(`/api/v1/customer/search?q=${encodeURIComponent(catalogSearchQuery.value)}`, { credentials: 'same-origin' });
    if (res.ok) {
      const data = await res.json();
      catalogSearchResults.value = data.hits || [];
      showSearchDropdown.value = true;
    }
  } catch (e) {
    console.error('search error', e);
  }
};

const selectCatalogResult = (item) => {
  showSearchDropdown.value = false;
  if (item.category === 'fund' || item.category === 'trading') {
    currentTab.value = 'trading';
  } else if (item.category === 'transparency' || item.category === 'welfare') {
    currentTab.value = 'esg';
  } else if (item.category === 'statement') {
    currentTab.value = 'statements';
  }
};

// Support & Inquiries State
const customerTickets = ref([]);
const selectedTicket = ref(null);
const ticketComments = ref([]);
const ticketForm = ref({ category: 'Trading', title: '', body: '' });
const newCommentText = ref('');
const isSubmittingSupportTicket = ref(false);
const isSubmittingComment = ref(false);

const videoPlayer = ref(null);
const peerConnection = ref(null);
const streamStatus = ref('connecting');
const isAudioMuted = ref(true);
const selectedStreamId = ref('str-018f-0001');

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
  { stream_id: 'str-018f-0001', name: 'Alpha Sunbeam Lounge Cam 1', stream_url: 'https://stream.oci.local/webrtc/alpha-lounge-cam1', protocol: 'webrtc', status: 'active' },
  { stream_id: 'str-018f-0002', name: '4K Feline Perch Scope', stream_url: 'https://stream.oci.local/webrtc/4k-perch-cam2', protocol: 'webrtc', status: 'active' }
]);

const selectedStream = computed(() => {
  return streams.value.find(s => s.stream_id === selectedStreamId.value) || streams.value[0];
});

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

const apiKeys = ref([]);

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

// Statements & Tax Documentation State
const statements = ref([]);
const isGeneratingMonthly = ref(false);
const isGeneratingTax = ref(false);
const statementMonth = ref('03');
const statementYear = ref(2026);
const taxYear = ref(2025);

// Lifecycle & Privacy State
const showRegisterModal = ref(false);
const isRegistering = ref(false);
const registerForm = ref({
  name: '',
  email: '',
  password: '',
  bank_account: 'Chase Premier Checking (****4821)',
  initial_deposit: 0
});

const showCloseModal = ref(false);
const closeConfirmChecked = ref(false);
const isClosingAccount = ref(false);
const closureReceipt = ref(null);

const showGDPRModal = ref(false);
const gdprConfirmText = ref('');
const isErasingData = ref(false);
const erasureCertificate = ref(null);

const isPasswordValid = computed(() => {
  const p = registerForm.value.password || '';
  return p.length >= 8 && /[A-Z]/.test(p) && /[a-z]/.test(p) && /[0-9]/.test(p) && /[^A-Za-z0-9]/.test(p);
});

const passwordStrengthPct = computed(() => {
  const p = registerForm.value.password || '';
  let score = 0;
  if (p.length >= 8) score += 20;
  if (p.length >= 12) score += 20;
  if (/[A-Z]/.test(p)) score += 20;
  if (/[0-9]/.test(p)) score += 20;
  if (/[^A-Za-z0-9]/.test(p)) score += 20;
  return score;
});

const passwordStrengthLabel = computed(() => {
  const s = passwordStrengthPct.value;
  if (s >= 80) return 'Strong (NIST Compliant)';
  if (s >= 60) return 'Moderate';
  if (s >= 20) return 'Weak';
  return 'Too Short';
});

const passwordStrengthColor = computed(() => {
  const s = passwordStrengthPct.value;
  if (s >= 80) return '#4ade80';
  if (s >= 60) return '#facc15';
  return '#f87171';
});

function showToast(msg) {
  toastMessage.value = msg;
  setTimeout(() => { toastMessage.value = ''; }, 3500);
}

function selectStream(stream) {
  selectedStreamId.value = stream.stream_id;
  initWebRTCStream();
}

function toggleAudio() {
  isAudioMuted.value = !isAudioMuted.value;
  showToast(isAudioMuted.value ? 'Audio muted' : 'Audio stream unmuted (Purr frequency: 28Hz)');
}

function recenterCamera() {
  showToast('Camera angle adjusted to Alpha Perch Zone 1');
}

function stopWebRTCStream() {
  if (peerConnection.value) {
    peerConnection.value.close();
    peerConnection.value = null;
  }
}

async function initWebRTCStream() {
  streamStatus.value = 'connecting';
  stopWebRTCStream();

  try {
    const config = {
      iceServers: [
        { urls: 'stun:stun.l.google.com:19302' },
        { urls: 'stun:stun.oci.local:3478' }
      ]
    };

    const pc = new RTCPeerConnection(config);
    peerConnection.value = pc;

    pc.ontrack = (event) => {
      if (videoPlayer.value && event.streams && event.streams[0]) {
        videoPlayer.value.srcObject = event.streams[0];
        streamStatus.value = 'live';
      }
    };

    pc.onconnectionstatechange = () => {
      if (pc.connectionState === 'connected') {
        streamStatus.value = 'live';
      } else if (pc.connectionState === 'failed' || pc.connectionState === 'disconnected') {
        streamStatus.value = 'disconnected';
      }
    };

    pc.addTransceiver('video', { direction: 'recvonly' });
    pc.addTransceiver('audio', { direction: 'recvonly' });

    const offer = await pc.createOffer();
    await pc.setLocalDescription(offer);

    const streamUrl = selectedStream.value?.stream_url || '/api/v1/customer/webrtc/whep';
    const res = await fetch(streamUrl, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/sdp',
        'X-CSRF-Token': csrfToken.value
      },
      body: offer.sdp
    }).catch(() => null);

    if (res && res.ok) {
      const answerSdp = await res.text();
      await pc.setRemoteDescription(new RTCSessionDescription({ type: 'answer', sdp: answerSdp }));
      streamStatus.value = 'live';
    } else {
      setupMockVideoStream();
    }
  } catch (err) {
    setupMockVideoStream();
  }
}

function setupMockVideoStream() {
  if (!videoPlayer.value) return;
  try {
    const canvas = document.createElement('canvas');
    canvas.width = 1280;
    canvas.height = 720;
    const ctx = canvas.getContext('2d');
    let frame = 0;

    const draw = () => {
      frame++;
      ctx.fillStyle = '#0f172a';
      ctx.fillRect(0, 0, canvas.width, canvas.height);

      ctx.strokeStyle = '#1e293b';
      ctx.lineWidth = 1;
      for (let x = 0; x < canvas.width; x += 40) {
        ctx.beginPath(); ctx.moveTo(x, 0); ctx.lineTo(x, canvas.height); ctx.stroke();
      }
      for (let y = 0; y < canvas.height; y += 40) {
        ctx.beginPath(); ctx.moveTo(0, y); ctx.lineTo(canvas.width, y); ctx.stroke();
      }

      const catX = 640 + Math.sin(frame * 0.03) * 200;
      const catY = 360 + Math.cos(frame * 0.05) * 80;
      ctx.fillStyle = '#ea580c';
      ctx.beginPath();
      ctx.arc(catX, catY, 40, 0, Math.PI * 2);
      ctx.fill();

      ctx.font = '24px monospace';
      ctx.fillStyle = '#f8fafc';
      ctx.fillText('🐈 Garfield (High-Velocity Zoomies)', catX - 160, catY - 50);

      requestAnimationFrame(draw);
    };
    draw();

    if (canvas.captureStream) {
      videoPlayer.value.srcObject = canvas.captureStream(60);
      videoPlayer.value.play().catch(() => {});
      streamStatus.value = 'live';
    } else {
      streamStatus.value = 'live';
    }
  } catch (e) {
    streamStatus.value = 'live';
  }
}

async function loginSession(customerID, name, email) {
  try {
    const res = await fetch('/api/v1/customer/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ customer_id: customerID, name, email })
    });
    if (res.ok) {
      const data = await res.json();
      if (data.csrf_token) {
        csrfToken.value = data.csrf_token;
      }
    }
  } catch (e) {
    if (!csrfToken.value) csrfToken.value = 'oci-dev-csrf-token';
  }
}

async function fetchESGTransparency() {
  try {
    const res = await fetch('/api/v1/customer/transparency/welfare');
    if (res.ok) {
      const data = await res.json();
      if (data && data.overall_welfare_score !== undefined) {
        esgData.value = data;
      }
    }
  } catch (e) {}
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
  } catch (e) {}
}

async function fetchTradeLogs() {
  try {
    const res = await fetch('/api/v1/customer/trade-logs');
    if (res.ok) {
      const data = await res.json();
      if (Array.isArray(data) && data.length > 0) {
        tradeLogs.value = data;
      }
    }
  } catch (e) {}
}

async function fetchStreams() {
  try {
    const res = await fetch('/api/v1/customer/streams');
    if (res.ok) {
      const data = await res.json();
      if (Array.isArray(data) && data.length > 0) {
        streams.value = data;
      }
    }
  } catch (e) {}
}

async function fetchDeposits() {
  try {
    const res = await fetch('/api/v1/customer/deposits');
    if (res.ok) {
      const data = await res.json();
      if (Array.isArray(data) && data.length > 0) {
        deposits.value = data;
      }
    }
  } catch (e) {}
}

async function fetchAPIKeys() {
  try {
    const res = await fetch('/api/v1/customer/api-keys');
    if (res.ok) {
      const data = await res.json();
      if (Array.isArray(data) && data.length > 0) {
        apiKeys.value = data;
      }
    }
  } catch (e) {}
}

async function fetchWebhooks() {
  try {
    const res = await fetch('/api/v1/customer/webhooks');
    if (res.ok) {
      const data = await res.json();
      if (Array.isArray(data) && data.length > 0) {
        webhooks.value = data;
      }
    }
  } catch (e) {}
}

async function fetchZoomieIndex() {
  try {
    const res = await fetch('/api/v1/customer/zoomie-index');
    if (res.ok) {
      const data = await res.json();
      if (data && data.zoomie_index_score !== undefined) {
        zoomieIndex.value = data;
      }
    }
  } catch (e) {}
}

async function fetchTickerData() {
  try {
    const res = await fetch('/api/v1/customer/ticker');
    if (res.ok) {
      const data = await res.json();
      if (Array.isArray(data) && data.length > 0) {
        tickerData.value = data;
      }
    }
  } catch (e) {}
}

async function togglePersona() {
  if (currentPersona.value === 'arthur') {
    currentPersona.value = 'chloe';
    currentTab.value = 'trading';
    await loginSession('018e0000-0000-7000-8000-000000000098', 'Chloe Spark', 'chloe@oci.local');
    fetchPortfolio();
    fetchTradeLogs();
    selectedTicket.value = null;
    fetchCustomerTickets();
    fetchStatements();
    showToast('Switched to Chloe Spark (Momentum Trader mode)');
  } else {
    currentPersona.value = 'arthur';
    currentTab.value = 'portfolio';
    await loginSession('018e0000-0000-7000-8000-000000000099', 'Arthur Pendelton', 'arthur@oci.local');
    fetchPortfolio();
    fetchTradeLogs();
    selectedTicket.value = null;
    fetchCustomerTickets();
    fetchStatements();
    showToast('Switched to Arthur Pendelton (Long-Term Investor mode)');
  }
}

async function executeQuickTrade() {
  showToast('⚡ Executing 1-Click Zoomie Pounce Order for 25 NVDA shares...');
  try {
    const res = await fetch('/api/v1/customer/trading/orders', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-CSRF-Token': csrfToken.value
      },
      body: JSON.stringify({
        portfolio_id: portfolio.value?.portfolio_id,
        symbol: 'NVDA',
        side: 'buy',
        quantity: 25.0,
        price: 138.80,
        order_type: 'market'
      })
    });
    if (!res.ok) {
      const err = await res.json().catch(() => ({}));
      throw new Error(err.error || `HTTP ${res.status}`);
    }
    const created = await res.json();
    tradeLogs.value.unshift(created);
    showToast('⚡ 1-Click Zoomie Pounce Order filled!');
  } catch (err) {
    showToast(`Quick trade failed: ${err.message}`);
  }
}

async function submitOrder() {
  isSubmittingOrder.value = true;
  try {
    const res = await fetch('/api/v1/customer/trading/orders', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-CSRF-Token': csrfToken.value
      },
      body: JSON.stringify({
        portfolio_id: portfolio.value?.portfolio_id,
        symbol: orderForm.value.symbol,
        side: orderForm.value.side,
        quantity: parseFloat(orderForm.value.quantity),
        price: 224.30,
        order_type: 'market'
      })
    });
    if (!res.ok) {
      const err = await res.json().catch(() => ({}));
      throw new Error(err.error || `HTTP ${res.status}`);
    }
    const created = await res.json();
    tradeLogs.value.unshift(created);
    showToast(`Order executed: ${orderForm.value.side.toUpperCase()} ${orderForm.value.quantity} ${orderForm.value.symbol}`);
  } catch (err) {
    showToast(`Order failed: ${err.message}`);
  } finally {
    isSubmittingOrder.value = false;
  }
}

async function submitDeposit() {
  isSubmittingDeposit.value = true;
  try {
    const res = await fetch('/api/v1/customer/deposits', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-CSRF-Token': csrfToken.value
      },
      body: JSON.stringify({
        amount_usd: parseFloat(depositForm.value.amount_usd),
        frequency: depositForm.value.frequency,
        day_of_month: 1,
        bank_account: depositForm.value.bank_account
      })
    });
    if (!res.ok) {
      const err = await res.json().catch(() => ({}));
      throw new Error(err.error || `HTTP ${res.status}`);
    }
    const created = await res.json();
    deposits.value.unshift(created);
    showToast(`Recurring ACH deposit scheduled for $${depositForm.value.amount_usd}`);
  } catch (err) {
    showToast(`Deposit failed: ${err.message}`);
  } finally {
    isSubmittingDeposit.value = false;
  }
}

async function createAPIKey() {
  try {
    const res = await fetch('/api/v1/customer/api-keys', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-CSRF-Token': csrfToken.value
      },
      body: JSON.stringify({
        name: apiKeyForm.value.name,
        permissions: ['read_streams', 'execute_orders', 'read_telemetry']
      })
    });
    if (!res.ok) {
      const err = await res.json().catch(() => ({}));
      throw new Error(err.error || `HTTP ${res.status}`);
    }
    const created = await res.json();
    apiKeys.value.unshift(created);
    newlyCreatedKey.value = created.api_key;
    showToast(`API Key generated for ${apiKeyForm.value.name}`);
  } catch (err) {
    showToast(`API Key generation failed: ${err.message}`);
  }
}

async function createWebhook() {
  try {
    const res = await fetch('/api/v1/customer/webhooks', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-CSRF-Token': csrfToken.value
      },
      body: JSON.stringify({
        target_url: webhookForm.value.target_url,
        events: ['events.observation.cat_spotted.v1', 'events.trade.executed.v1']
      })
    });
    if (!res.ok) {
      const err = await res.json().catch(() => ({}));
      throw new Error(err.error || `HTTP ${res.status}`);
    }
    const created = await res.json();
    webhooks.value.unshift(created);
    showToast(`Webhook subscribed to ${webhookForm.value.target_url}`);
  } catch (err) {
    showToast(`Webhook subscription failed: ${err.message}`);
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

function formatDate(dt) {
  if (!dt) return '';
  const d = new Date(dt);
  return isNaN(d.getTime()) ? dt : d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' });
}

async function fetchCustomerTickets() {
  try {
    const res = await fetch('/api/v1/customer/support/tickets');
    if (res.ok) {
      const data = await res.json();
      customerTickets.value = data.items || [];
      if (selectedTicket.value) {
        const found = customerTickets.value.find(t => t.ticket_id === selectedTicket.value.ticket_id);
        if (found) selectedTicket.value = found;
      }
    }
  } catch (e) {}
}

async function fetchTicketComments(ticketId) {
  try {
    const res = await fetch(`/api/v1/customer/support/tickets/${ticketId}/comments`);
    if (res.ok) {
      const data = await res.json();
      ticketComments.value = data.items || [];
    }
  } catch (e) {}
}

function selectCustomerTicket(ticket) {
  selectedTicket.value = ticket;
  ticketComments.value = [];
  fetchTicketComments(ticket.ticket_id);
}

async function submitCustomerTicket() {
  if (!ticketForm.value.title) {
    showToast('Inquiry subject is required.');
    return;
  }
  isSubmittingSupportTicket.value = true;
  try {
    const res = await fetch('/api/v1/customer/support/tickets', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-CSRF-Token': csrfToken.value
      },
      body: JSON.stringify({
        category: ticketForm.value.category,
        title: ticketForm.value.title,
        body: ticketForm.value.body
      })
    });
    if (res.ok) {
      const created = await res.json();
      customerTickets.value.unshift(created);
      selectedTicket.value = created;
      ticketComments.value = [];
      showToast('Support inquiry submitted! We will respond shortly.');
      ticketForm.value = { category: 'Trading', title: '', body: '' };
    } else {
      const err = await res.json().catch(() => ({}));
      showToast(err.error || 'Failed to submit inquiry.');
    }
  } catch (e) {
    showToast('Failed to submit inquiry.');
  } finally {
    isSubmittingSupportTicket.value = false;
  }
}

async function submitTicketComment() {
  if (!newCommentText.value || !selectedTicket.value) return;
  isSubmittingComment.value = true;
  try {
    const res = await fetch(`/api/v1/customer/support/tickets/${selectedTicket.value.ticket_id}/comments`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-CSRF-Token': csrfToken.value
      },
      body: JSON.stringify({
        comment: newCommentText.value
      })
    });
    if (res.ok) {
      const created = await res.json();
      ticketComments.value.push(created);
      newCommentText.value = '';
      showToast('Reply sent to support desk.');
    } else {
      showToast('Failed to send reply.');
    }
  } catch (e) {
    showToast('Failed to send reply.');
  } finally {
    isSubmittingComment.value = false;
  }
}

async function fetchStatements() {
  try {
    const res = await fetch('/api/v1/customer/statements');
    if (res.ok) {
      const data = await res.json();
      statements.value = data.items || [];
    }
  } catch (e) {}
}

async function generateMonthlyStatementWorkflow() {
  isGeneratingMonthly.value = true;
  try {
    const res = await fetch('/api/v1/customer/statements/generate', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-CSRF-Token': csrfToken.value
      },
      body: JSON.stringify({
        type: 'monthly',
        period_month: parseInt(statementMonth.value),
        period_year: parseInt(statementYear.value)
      })
    });
    if (res.ok) {
      const created = await res.json();
      statements.value.unshift(created);
      showToast('Monthly Statement PDF generated successfully!');
    } else {
      const err = await res.json().catch(() => ({}));
      showToast(err.error || 'Failed to generate monthly statement');
    }
  } catch (e) {
    showToast('Failed to generate monthly statement');
  } finally {
    isGeneratingMonthly.value = false;
  }
}

async function generateForm1099BWorkflow() {
  isGeneratingTax.value = true;
  try {
    const res = await fetch('/api/v1/customer/statements/generate', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-CSRF-Token': csrfToken.value
      },
      body: JSON.stringify({
        type: '1099_b',
        tax_year: parseInt(taxYear.value)
      })
    });
    if (res.ok) {
      const created = await res.json();
      statements.value.unshift(created);
      showToast('Form 1099-B Tax Return generated successfully!');
    } else {
      const err = await res.json().catch(() => ({}));
      showToast(err.error || 'Failed to generate Form 1099-B');
    }
  } catch (e) {
    showToast('Failed to generate Form 1099-B');
  } finally {
    isGeneratingTax.value = false;
  }
}

function downloadStatement(stmt) {
  const url = `/api/v1/customer/statements/${stmt.statement_id}/download`;
  window.open(url, '_blank');
}

function downloadGDPRCertificate(certId) {
  const url = `/api/v1/customer/privacy/gdpr-certificate/${certId}/download`;
  window.open(url, '_blank');
}

async function handleRegisterCustomer() {
  if (!isPasswordValid.value) {
    showToast('Password does not meet NIST complexity criteria.');
    return;
  }
  isRegistering.value = true;
  try {
    const res = await fetch('/api/v1/customer/auth/register', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        name: registerForm.value.name,
        email: registerForm.value.email,
        password: registerForm.value.password,
        bank_account: registerForm.value.bank_account,
        initial_deposit_usd: registerForm.value.initial_deposit || 0
      })
    });
    if (res.ok) {
      const data = await res.json();
      showToast(`Account successfully registered! Welcome, ${data.name}`);
      showRegisterModal.value = false;
      registerForm.value = { name: '', email: '', password: '', bank_account: 'Chase Premier Checking (****4821)', initial_deposit: 0 };
      if (data.portfolio && data.portfolio.portfolio_id) {
        portfolio.value = data.portfolio;
      }
      fetchPortfolio();
      fetchStatements();
    } else {
      const err = await res.json().catch(() => ({}));
      showToast(err.error || 'Failed to register account');
    }
  } catch (e) {
    showToast('Failed to register account');
  } finally {
    isRegistering.value = false;
  }
}

async function handleCloseAccountWorkflow() {
  isClosingAccount.value = true;
  try {
    const res = await fetch('/api/v1/customer/account/close', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-CSRF-Token': csrfToken.value
      },
      body: JSON.stringify({
        portfolio_id: portfolio.value?.portfolio_id,
        reason: 'Customer initiated self-service liquidation'
      })
    });
    if (res.ok) {
      const data = await res.json();
      closureReceipt.value = data;
      showCloseModal.value = false;
      closeConfirmChecked.value = false;
      showToast('Portfolio liquidated and account closed. Funds disbursed via ACH.');
      fetchPortfolio();
    } else {
      const err = await res.json().catch(() => ({}));
      showToast(err.error || 'Failed to close account');
    }
  } catch (e) {
    showToast('Failed to close account');
  } finally {
    isClosingAccount.value = false;
  }
}

async function handleGDPRErasureWorkflow() {
  if (gdprConfirmText.value !== 'DELETE') return;
  isErasingData.value = true;
  try {
    const res = await fetch('/api/v1/customer/privacy/gdpr-erasure', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-CSRF-Token': csrfToken.value
      },
      body: JSON.stringify({
        confirmation: 'DELETE',
        reason: 'GDPR Article 17 Right to be Forgotten Request'
      })
    });
    if (res.ok) {
      const data = await res.json();
      erasureCertificate.value = data;
      showGDPRModal.value = false;
      gdprConfirmText.value = '';
      showToast('GDPR Erasure completed. Personal data permanently scrubbed.');
    } else {
      const err = await res.json().catch(() => ({}));
      showToast(err.error || 'Failed to execute GDPR erasure');
    }
  } catch (e) {
    showToast('Failed to execute GDPR erasure');
  } finally {
    isErasingData.value = false;
  }
}

watch(currentTab, (newTab) => {
  if (newTab === 'stream') {
    initWebRTCStream();
  } else {
    stopWebRTCStream();
  }
  if (newTab === 'statements') {
    fetchStatements();
  }
});

onMounted(async () => {
  if (currentPersona.value === 'chloe') {
    await loginSession('018e0000-0000-7000-8000-000000000098', 'Chloe Spark', 'chloe@oci.local');
  } else {
    await loginSession('018e0000-0000-7000-8000-000000000099', 'Arthur Pendelton', 'arthur@oci.local');
  }
  fetchPortfolio();
  fetchTradeLogs();
  fetchStreams();
  fetchDeposits();
  fetchAPIKeys();
  fetchWebhooks();
  fetchESGTransparency();
  fetchZoomieIndex();
  fetchTickerData();
  fetchCustomerTickets();
  fetchStatements();
  if (currentTab.value === 'stream') {
    initWebRTCStream();
  }
});

onUnmounted(() => {
  stopWebRTCStream();
});
</script>

<style scoped>
.container { max-width: 1100px; margin: 0 auto; color: var(--text-color, #f8fafc); font-family: var(--font-family, sans-serif); }
header { display: flex; justify-content: space-between; align-items: center; border-bottom: 1px solid #334155; padding-bottom: 1rem; margin-bottom: 1.5rem; }
.brand { display: flex; align-items: center; gap: 0.75rem; }
.brand-title h1 { margin: 0; color: var(--secondary-color, #ea580c); font-size: 1.5rem; }
.brand-title p { margin: 0; color: #94a3b8; font-size: 0.85rem; }
.search-bar-container { position: relative; flex: 1; max-width: 380px; margin: 0 1rem; }
.search-bar-container input { width: 100%; background: var(--card-color, #1e293b); border: 1px solid #334155; border-radius: 9999px; padding: 0.5rem 1rem 0.5rem 2.25rem; color: #f8fafc; font-size: 0.85rem; outline: none; transition: border-color 0.2s; }
.search-bar-container input:focus { border-color: var(--secondary-color, #ea580c); }
.search-bar-container .search-icon { position: absolute; left: 0.75rem; top: 50%; transform: translateY(-50%); font-size: 0.85rem; color: #94a3b8; pointer-events: none; }
.search-dropdown-menu { position: absolute; top: calc(100% + 0.5rem); left: 0; right: 0; background: var(--card-color, #1e293b); border: 1px solid #475569; border-radius: 0.5rem; box-shadow: 0 10px 25px -5px rgba(0,0,0,0.5); max-height: 280px; overflow-y: auto; z-index: 50; padding: 0.25rem; }
.search-dropdown-item { display: flex; align-items: flex-start; gap: 0.5rem; padding: 0.5rem 0.75rem; border-radius: 0.35rem; cursor: pointer; transition: background 0.15s; }
.search-dropdown-item:hover { background: #334155; }
.item-tag { font-size: 0.65rem; font-weight: bold; padding: 2px 5px; border-radius: 3px; background: #475569; color: white; }
.item-tag.fund { background: #ea580c; }
.item-tag.transparency { background: #059669; }
.item-tag.statement { background: #3b82f6; }
.item-text strong { display: block; font-size: 0.85rem; color: #f8fafc; }
.item-text small { color: #94a3b8; font-size: 0.75rem; }

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

.video-player-container { background: var(--background-color, #0f172a); border-radius: 0.5rem; height: 320px; display: flex; align-items: center; justify-content: center; position: relative; border: 1px solid #334155; overflow: hidden; }
.video-wrapper { width: 100%; height: 100%; position: relative; display: flex; align-items: center; justify-content: center; }
.video-element { width: 100%; height: 100%; object-fit: cover; border-radius: 0.5rem; background: #000; }
.video-overlay { position: absolute; bottom: 1rem; left: 1rem; right: 1rem; background: rgba(15, 23, 42, 0.75); backdrop-filter: blur(4px); padding: 0.75rem; border-radius: 0.5rem; border: 1px solid #334155; text-align: center; }
.stream-error-overlay { position: absolute; text-align: center; color: #f87171; z-index: 5; }
.stream-card.active { border-color: var(--secondary-color, #ea580c); background: rgba(234, 88, 12, 0.1); }
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

.badge { display: inline-block; padding: 0.2rem 0.5rem; border-radius: 0.25rem; font-size: 0.75rem; font-weight: bold; }
.badge-open { background: #0284c7; color: white; }
.badge-ack { background: #d97706; color: white; }
.badge-closed { background: #059669; color: white; }

.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.75);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: 1rem;
}
.modal-card {
  background: var(--card-color, #1e293b);
  border: 1px solid #334155;
  border-radius: 0.75rem;
  max-width: 540px;
  width: 100%;
  padding: 1.5rem;
  box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.5);
  max-height: 90vh;
  overflow-y: auto;
}
</style>
