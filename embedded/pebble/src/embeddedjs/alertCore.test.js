import { test } from 'node:test';
import assert from 'node:assert/strict';
import { AlertQueue, CMD_ALERT, CMD_ACK_CONFIRM } from './alertCore.js';

test('AlertQueue - process incoming alert AppMessage', () => {
  const queue = new AlertQueue();
  const rawMsg = {
    CMD: CMD_ALERT,
    TICKET_ID: 'ticket-uuid-001',
    TITLE: 'K3s Node Disk Pressure Warning',
    BODY: 'Node oci-k3s-worker-01 disk usage above 85%',
    PRIORITY: 'HIGH',
    AUTHOR: 'frank_ops',
    TIMESTAMP: '2025-01-15T10:00:00Z',
  };

  const alert = queue.processAppMessage(rawMsg);
  assert.notEqual(alert, null);
  assert.equal(alert.ticketId, 'ticket-uuid-001');
  assert.equal(alert.title, 'K3s Node Disk Pressure Warning');
  assert.equal(alert.acknowledged, false);

  const pending = queue.getPendingAlerts();
  assert.equal(pending.length, 1);
});

test('AlertQueue - acknowledge alert and generate ACK payload', () => {
  const queue = new AlertQueue();
  queue.processAppMessage({
    CMD: CMD_ALERT,
    TICKET_ID: 'ticket-uuid-002',
    TITLE: 'Certificate Expiration Warning',
    BODY: 'Intermediate CA expires in 5 days',
  });

  const ackPayload = queue.acknowledgeAlert('ticket-uuid-002');
  assert.equal(ackPayload.CMD, CMD_ACK_CONFIRM);
  assert.equal(ackPayload.TICKET_ID, 'ticket-uuid-002');

  const pending = queue.getPendingAlerts();
  assert.equal(pending.length, 0);

  // Subsequent alert with same ticketId should be ignored
  const secondTry = queue.processAppMessage({
    CMD: CMD_ALERT,
    TICKET_ID: 'ticket-uuid-002',
    TITLE: 'Certificate Expiration Warning',
  });
  assert.equal(secondTry, null);
});
