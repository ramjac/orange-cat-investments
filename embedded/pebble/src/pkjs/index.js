/**
 * index.js - PebbleKit JS (Phone-side) Gateway Bridge for OCI On-Call Watch App
 */

const PROXY_URL = 'http://localhost:8082';

function fetchAlerts() {
  const xhr = new XMLHttpRequest();
  xhr.open('GET', PROXY_URL + '/ops/pebble/alerts', true);
  xhr.onload = function () {
    if (xhr.status === 200) {
      try {
        const alerts = JSON.parse(xhr.responseText);
        if (Array.isArray(alerts) && alerts.length > 0) {
          alerts.forEach((alert) => {
            if (alert.app_message_dict) {
              Pebble.sendAppMessage(alert.app_message_dict, function () {
                console.log('[PKJS] Sent alert AppMessage to watch:', alert.ticket_id);
              });
            }
          });
        }
      } catch (e) {
        console.error('[PKJS] Error parsing alert JSON response:', e);
      }
    }
  };
  xhr.send();
}

Pebble.addEventListener('ready', function () {
  console.log('[PKJS] PebbleKit JS bridge ready');
  fetchAlerts();
  setInterval(fetchAlerts, 15000);
});

Pebble.addEventListener('appmessage', function (e) {
  console.log('[PKJS] Received ACK AppMessage from watch:', JSON.stringify(e.payload));
  if (e.payload && e.payload.CMD === 2) {
    const xhr = new XMLHttpRequest();
    xhr.open('POST', PROXY_URL + '/ops/pebble/ack', true);
    xhr.setRequestHeader('Content-Type', 'application/json');
    xhr.send(JSON.stringify({
      ticket_id: e.payload.TICKET_ID,
      acknowledged_by: 'frank_ops_watch',
    }));
  }
});
