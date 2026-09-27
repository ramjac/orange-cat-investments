/**
 * main.js - Watch-side On-Call Alert App using RePebble Alloy (Moddable XS embedded JS)
 */

import { AlertQueue, CMD_ALERT } from './alertCore.js';

const queue = new AlertQueue();

// Initialize Watch App UI Window
let currentAlertIndex = 0;

function updateUI() {
  const pending = queue.getPendingAlerts();
  if (pending.length === 0) {
    console.log('[OCI-Alerts] All alerts clear. Standard operations active.');
    return;
  }

  if (currentAlertIndex >= pending.length) {
    currentAlertIndex = 0;
  }

  const alert = pending[currentAlertIndex];
  console.log(`[OCI-Alerts] Displaying Alert ${currentAlertIndex + 1}/${pending.length}:`);
  console.log(` Title:    ${alert.title}`);
  console.log(` Priority: ${alert.priority}`);
  console.log(` Author:   ${alert.author}`);
  console.log(` Body:     ${alert.body}`);
}

// AppMessage Handlers
function onAppMessageReceived(dict) {
  console.log('[OCI-Alerts] Received AppMessage dictionary');
  const alert = queue.processAppMessage(dict);
  if (alert) {
    console.log(`[OCI-Alerts] New alert queued: ${alert.title} (${alert.ticketId})`);
    updateUI();
  }
}

// Button Click Event Handlers
function onSelectClicked() {
  const pending = queue.getPendingAlerts();
  if (pending.length === 0) return;

  const currentAlert = pending[currentAlertIndex];
  console.log(`[OCI-Alerts] ACK button pressed for ticket ${currentAlert.ticketId}`);

  const ackMsg = queue.acknowledgeAlert(currentAlert.ticketId);

  // Send ACK AppMessage back to phone companion proxy
  if (typeof Pebble !== 'undefined' && Pebble.sendAppMessage) {
    Pebble.sendAppMessage(ackMsg, () => {
      console.log(`[OCI-Alerts] ACK successfully delivered to gateway proxy for ${currentAlert.ticketId}`);
    }, (err) => {
      console.log(`[OCI-Alerts] ACK delivery failed: ${JSON.stringify(err)}`);
    });
  }

  updateUI();
}

function onUpClicked() {
  const pending = queue.getPendingAlerts();
  if (pending.length <= 1) return;

  currentAlertIndex = (currentAlertIndex - 1 + pending.length) % pending.length;
  updateUI();
}

function onDownClicked() {
  const pending = queue.getPendingAlerts();
  if (pending.length <= 1) return;

  currentAlertIndex = (currentAlertIndex + 1) % pending.length;
  updateUI();
}

// Export initialization hook for Moddable XS / Alloy runtime
export default function main() {
  console.log('[OCI-Alerts] OCI On-Call Alert Watch App (Alloy Embedded JS) Initialized');

  if (typeof Pebble !== 'undefined') {
    Pebble.addEventListener('appmessage', (e) => {
      onAppMessageReceived(e.payload);
    });
  }
}

main();
