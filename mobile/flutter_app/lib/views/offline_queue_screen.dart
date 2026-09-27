import 'package:flutter/material.dart';
import '../models/maintenance_log.dart';
import '../services/sync_engine.dart';

class OfflineQueueScreen extends StatefulWidget {
  final SyncEngine syncEngine;

  const OfflineQueueScreen({
    super.key,
    required this.syncEngine,
  });

  @override
  State<OfflineQueueScreen> createState() => _OfflineQueueScreenState();
}

class _OfflineQueueScreenState extends State<OfflineQueueScreen> {
  List<MaintenanceLog> _logs = [];
  bool _isLoading = true;
  bool _isSyncing = false;

  @override
  void initState() {
    super.initState();
    _loadLogs();
  }

  Future<void> _loadLogs() async {
    setState(() => _isLoading = true);
    final logs = await widget.syncEngine.dbHelper.getAllLogs();
    setState(() {
      _logs = logs;
      _isLoading = false;
    });
  }

  Future<void> _triggerSync() async {
    setState(() => _isSyncing = true);
    final result = await widget.syncEngine.syncPendingLogs();
    await _loadLogs();
    setState(() => _isSyncing = false);

    if (!mounted) return;
    final message = result.isSuccess
        ? 'Successfully synced ${result.syncedCount} offline maintenance log(s)!'
        : (result.errorMessage ?? 'Sync failed.');

    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text(message)),
    );
  }

  @override
  Widget build(BuildContext context) {
    final pendingCount = _logs.where((l) => !l.synced).length;

    return Scaffold(
      appBar: AppBar(
        title: const Text('Offline Maintenance Queue'),
        backgroundColor: Colors.orange.shade800,
        foregroundColor: Colors.white,
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh),
            onPressed: _loadLogs,
          ),
        ],
      ),
      body: _isLoading
          ? const Center(child: CircularProgressIndicator())
          : Column(
              children: [
                Container(
                  color: Colors.orange.shade100,
                  padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
                  child: Row(
                    children: [
                      Icon(
                        pendingCount > 0 ? Icons.sync_problem : Icons.cloud_done,
                        color: pendingCount > 0 ? Colors.deepOrange : Colors.green,
                      ),
                      const SizedBox(width: 12),
                      Expanded(
                        child: Text(
                          '$pendingCount pending log(s) awaiting cloud synchronization.',
                          style: const TextStyle(fontWeight: FontWeight.bold),
                        ),
                      ),
                      ElevatedButton.icon(
                        onPressed: (_isSyncing || pendingCount == 0) ? null : _triggerSync,
                        style: ElevatedButton.styleFrom(
                          backgroundColor: Colors.orange.shade800,
                          foregroundColor: Colors.white,
                        ),
                        icon: _isSyncing
                            ? const SizedBox(
                                width: 16,
                                height: 16,
                                child: CircularProgressIndicator(strokeWidth: 2, color: Colors.white),
                              )
                            : const Icon(Icons.sync, size: 18),
                        label: const Text('Sync Now'),
                      ),
                    ],
                  ),
                ),
                Expanded(
                  child: _logs.isEmpty
                      ? const Center(
                          child: Text(
                            'No maintenance logs found in local SQLite database.',
                            style: TextStyle(color: Colors.grey),
                          ),
                        )
                      : ListView.builder(
                          itemCount: _logs.length,
                          itemBuilder: (context, index) {
                            final log = _logs[index];
                            return Card(
                              margin: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
                              child: ListTile(
                                leading: CircleAvatar(
                                  backgroundColor: log.synced ? Colors.green.shade100 : Colors.orange.shade100,
                                  child: Icon(
                                    log.synced ? Icons.cloud_done : Icons.cloud_off,
                                    color: log.synced ? Colors.green : Colors.deepOrange,
                                  ),
                                ),
                                title: Text(log.actionTaken, style: const TextStyle(fontWeight: FontWeight.bold)),
                                subtitle: Text('QR: ${log.qrCodeScanned}\nAsset: ${log.assetId}'),
                                trailing: Column(
                                  mainAxisAlignment: MainAxisAlignment.center,
                                  crossAxisAlignment: CrossAxisAlignment.end,
                                  children: [
                                    Chip(
                                      label: Text(
                                        log.synced ? 'Synced' : 'Pending',
                                        style: TextStyle(
                                          fontSize: 10,
                                          color: log.synced ? Colors.green.shade900 : Colors.deepOrange.shade900,
                                        ),
                                      ),
                                      backgroundColor: log.synced ? Colors.green.shade50 : Colors.orange.shade50,
                                      padding: EdgeInsets.zero,
                                    ),
                                    Text(
                                      '${log.createdAt.hour}:${log.createdAt.minute.toString().padLeft(2, '0')}',
                                      style: const TextStyle(fontSize: 10, color: Colors.grey),
                                    ),
                                  ],
                                ),
                              ),
                            );
                          },
                        ),
                ),
              ],
            ),
    );
  }
}
