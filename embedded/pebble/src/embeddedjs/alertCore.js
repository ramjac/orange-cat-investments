/**
 * alertCore.js - Platform-independent domain logic for OCI On-Call Watch Alerts
 * Executed within the RePebble Alloy (Moddable XS) runtime or Node.js test environment.
 */

export const CMD_ALERT = 1;
export const CMD_ACK_CONFIRM = 2;

export class AlertQueue {
  constructor() {
    this.alerts = [];
    this.acknowledgedTicketIds = new Set();
  }

  /**
   * Process an incoming AppMessage dictionary payload from Pebble gateway proxy
   * @param {Object} msg AppMessage dictionary
   * @returns {Object|null} Formatted alert object or null if ignored
   */
  processAppMessage(msg) {
    if (!msg || msg.CMD !== CMD_ALERT) {
      return null;
    }

    const ticketId = msg.TICKET_ID || msg.ticket_id;
    if (!ticketId || this.acknowledgedTicketIds.has(ticketId)) {
      return null;
    }

    const alert = {
      ticketId,
      title: msg.TITLE || msg.title || 'Untitled Alert',
      body: msg.BODY || msg.body || '',
      priority: msg.PRIORITY || msg.priority || 'HIGH',
      author: msg.AUTHOR || msg.author || 'system',
      timestamp: msg.TIMESTAMP || msg.timestamp || new Date().toISOString(),
      receivedAt: new Date().toISOString(),
      acknowledged: false,
    };

    // Replace existing alert with same ticketId or append new one
    const idx = this.alerts.findIndex((a) => a.ticketId === ticketId);
    if (idx >= 0) {
      this.alerts[idx] = alert;
    } else {
      this.alerts.push(alert);
    }

    return alert;
  }

  /**
   * Acknowledge an alert by ticket ID
   * @param {string} ticketId
   * @returns {Object} ACK payload for AppMessage transmission
   */
  acknowledgeAlert(ticketId) {
    this.acknowledgedTicketIds.add(ticketId);
    const alert = this.alerts.find((a) => a.ticketId === ticketId);
    if (alert) {
      alert.acknowledged = true;
    }

    return {
      CMD: CMD_ACK_CONFIRM,
      TICKET_ID: ticketId,
      TIMESTAMP: new Date().toISOString(),
    };
  }

  /**
   * Get active unacknowledged alerts
   */
  getPendingAlerts() {
    return this.alerts.filter((a) => !a.acknowledged);
  }

  /**
   * Clear acknowledged alerts
   */
  clearAcknowledged() {
    this.alerts = this.alerts.filter((a) => !a.acknowledged);
  }
}
