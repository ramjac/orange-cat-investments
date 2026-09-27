import 'package:flutter/material.dart';
import '../services/sync_engine.dart';
import '../services/api_service.dart';
import 'qr_scanner_screen.dart';
import 'maintenance_log_form.dart';
import 'offline_queue_screen.dart';
import 'firmware_ota_screen.dart';

class HomeScreen extends StatefulWidget {
  final SyncEngine syncEngine;
  final ApiService apiService;

  const HomeScreen({
    super.key,
    required this.syncEngine,
    required this.apiService,
  });

  @override
  State<HomeScreen> createState() => _HomeScreenState();
}

class _HomeScreenState extends State<HomeScreen> {
  int _pendingCount = 0;

  @override
  void initState() {
    super.initState();
    _updatePendingCount();
  }

  Future<void> _updatePendingCount() async {
    final unsynced = await widget.syncEngine.dbHelper.getUnsyncedLogs();
    setState(() {
      _pendingCount = unsynced.length;
    });
  }

  Future<void> _scanQRCodeAndLog() async {
    final qrResult = await Navigator.of(context).push<String>(
      MaterialPageRoute(builder: (_) => const QRScannerScreen()),
    );
    if (!mounted) return;
    if (qrResult != null && qrResult.isNotEmpty) {
      await Navigator.of(context).push(
        MaterialPageRoute(
          builder: (_) => MaintenanceLogFormScreen(
            syncEngine: widget.syncEngine,
            initialQrCode: qrResult,
          ),
        ),
      );
      _updatePendingCount();
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('OCI Field Maintenance'),
        backgroundColor: Colors.orange.shade800,
        foregroundColor: Colors.white,
      ),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(16.0),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Card(
              color: Colors.grey.shade900,
              child: Padding(
                padding: const EdgeInsets.all(16.0),
                child: Row(
                  children: [
                    const CircleAvatar(
                      radius: 28,
                      backgroundColor: Colors.orange,
                      child: Icon(Icons.person, color: Colors.white, size: 32),
                    ),
                    const SizedBox(width: 16),
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: const [
                          Text(
                            'Bob Builder',
                            style: TextStyle(
                              color: Colors.white,
                              fontSize: 18,
                              fontWeight: FontWeight.bold,
                            ),
                          ),
                          Text(
                            'Lead Facilities & Edge Telemetry Engineer',
                            style: TextStyle(color: Colors.orangeAccent, fontSize: 13),
                          ),
                          Text(
                            'Habitat Infrastructure Dept',
                            style: TextStyle(color: Colors.white70, fontSize: 12),
                          ),
                        ],
                      ),
                    ),
                  ],
                ),
              ),
            ),
            const SizedBox(height: 16),
            Card(
              elevation: 2,
              child: ListTile(
                leading: CircleAvatar(
                  backgroundColor: _pendingCount > 0 ? Colors.orange.shade100 : Colors.green.shade100,
                  child: Icon(
                    _pendingCount > 0 ? Icons.sync_problem : Icons.check_circle,
                    color: _pendingCount > 0 ? Colors.deepOrange : Colors.green,
                  ),
                ),
                title: const Text('Local SQLite Sync Status'),
                subtitle: Text('$_pendingCount log(s) pending offline sync'),
                trailing: ElevatedButton(
                  onPressed: () async {
                    await widget.syncEngine.syncPendingLogs();
                    _updatePendingCount();
                  },
                  child: const Text('Sync'),
                ),
              ),
            ),
            const SizedBox(height: 20),
            const Text(
              'Field Operations',
              style: TextStyle(fontSize: 16, fontWeight: FontWeight.bold),
            ),
            const SizedBox(height: 12),
            GridView.count(
              crossAxisCount: 2,
              shrinkWrap: true,
              crossAxisSpacing: 12,
              mainAxisSpacing: 12,
              physics: const NeverScrollableScrollPhysics(),
              children: [
                _buildOptionCard(
                  icon: Icons.qr_code_scanner,
                  title: 'Scan QR Code',
                  subtitle: 'Scan edge asset',
                  color: Colors.orange,
                  onTap: _scanQRCodeAndLog,
                ),
                _buildOptionCard(
                  icon: Icons.note_add,
                  title: 'Log Maintenance',
                  subtitle: 'New offline log',
                  color: Colors.blue,
                  onTap: () async {
                    await Navigator.of(context).push(
                      MaterialPageRoute(
                        builder: (_) => MaintenanceLogFormScreen(syncEngine: widget.syncEngine),
                      ),
                    );
                    _updatePendingCount();
                  },
                ),
                _buildOptionCard(
                  icon: Icons.sync,
                  title: 'Offline Queue',
                  subtitle: 'View local SQLite logs',
                  color: Colors.purple,
                  onTap: () async {
                    await Navigator.of(context).push(
                      MaterialPageRoute(
                        builder: (_) => OfflineQueueScreen(syncEngine: widget.syncEngine),
                      ),
                    );
                    _updatePendingCount();
                  },
                ),
                _buildOptionCard(
                  icon: Icons.system_update_alt,
                  title: 'Firmware OTA',
                  subtitle: 'Edge OTA status',
                  color: Colors.teal,
                  onTap: () {
                    Navigator.of(context).push(
                      MaterialPageRoute(
                        builder: (_) => FirmwareOTAScreen(apiService: widget.apiService),
                      ),
                    );
                  },
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildOptionCard({
    required IconData icon,
    required String title,
    required String subtitle,
    required Color color,
    required VoidCallback onTap,
  }) {
    return Card(
      elevation: 2,
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(12),
        child: Padding(
          padding: const EdgeInsets.all(16.0),
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              CircleAvatar(
                backgroundColor: color.withOpacity(0.15),
                radius: 24,
                child: Icon(icon, color: color, size: 28),
              ),
              const SizedBox(height: 8),
              Text(
                title,
                textAlign: TextAlign.center,
                style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 14),
              ),
              const SizedBox(height: 4),
              Text(
                subtitle,
                textAlign: TextAlign.center,
                style: const TextStyle(color: Colors.grey, fontSize: 11),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
