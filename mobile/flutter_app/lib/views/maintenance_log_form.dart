import 'package:flutter/material.dart';
import 'package:uuid/uuid.dart';
import '../models/maintenance_log.dart';
import '../services/sync_engine.dart';
import 'qr_scanner_screen.dart';

class MaintenanceLogFormScreen extends StatefulWidget {
  final SyncEngine syncEngine;
  final String? initialQrCode;

  const MaintenanceLogFormScreen({
    super.key,
    required this.syncEngine,
    this.initialQrCode,
  });

  @override
  State<MaintenanceLogFormScreen> createState() => _MaintenanceLogFormScreenState();
}

class _MaintenanceLogFormScreenState extends State<MaintenanceLogFormScreen> {
  final _formKey = GlobalKey<FormState>();
  final _uuid = const Uuid();

  late TextEditingController _qrCodeController;
  late TextEditingController _assetIdController;
  late TextEditingController _ticketIdController;
  late TextEditingController _actionTakenController;
  late TextEditingController _notesController;

  bool _isSaving = false;

  @override
  void initState() {
    super.initState();
    _qrCodeController = TextEditingController(text: widget.initialQrCode ?? 'QR-CAM-ORANGE-01');
    _assetIdController = TextEditingController(text: 'asset-uuid-001');
    _ticketIdController = TextEditingController();
    _actionTakenController = TextEditingController(text: 'Routine lens cleaning and sensor calibration');
    _notesController = TextEditingController();
  }

  @override
  void dispose() {
    _qrCodeController.dispose();
    _assetIdController.dispose();
    _ticketIdController.dispose();
    _actionTakenController.dispose();
    _notesController.dispose();
    super.dispose();
  }

  Future<void> _scanQRCode() async {
    final result = await Navigator.of(context).push<String>(
      MaterialPageRoute(builder: (_) => const QRScannerScreen()),
    );
    if (result != null && result.isNotEmpty) {
      setState(() {
        _qrCodeController.text = result;
      });
    }
  }

  Future<void> _saveLog({bool syncImmediately = false}) async {
    if (!_formKey.currentState!.validate()) return;

    setState(() => _isSaving = true);

    final log = MaintenanceLog(
      id: _uuid.v4(),
      assetId: _assetIdController.text.trim(),
      ticketId: _ticketIdController.text.trim().isEmpty ? null : _ticketIdController.text.trim(),
      technicianId: 'emp-human-bob',
      qrCodeScanned: _qrCodeController.text.trim(),
      actionTaken: _actionTakenController.text.trim(),
      notes: _notesController.text.trim().isEmpty ? null : _notesController.text.trim(),
      createdAt: DateTime.now().toUtc(),
    );

    await widget.syncEngine.saveLogLocally(log);

    String message = 'Maintenance log saved locally to SQLite offline queue.';
    if (syncImmediately) {
      final syncResult = await widget.syncEngine.syncPendingLogs();
      if (syncResult.isSuccess) {
        message = 'Maintenance log saved and synced to cloud successfully!';
      } else {
        message = 'Log saved locally (Offline mode). Sync will retry when connection is restored.';
      }
    }

    if (!mounted) return;
    setState(() => _isSaving = false);

    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text(message)),
    );

    Navigator.of(context).pop(true);
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Log Field Maintenance'),
        backgroundColor: Colors.orange.shade800,
        foregroundColor: Colors.white,
      ),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(20.0),
        child: Form(
          key: _formKey,
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Card(
                color: Colors.orange.shade50,
                child: Padding(
                  padding: const EdgeInsets.all(12.0),
                  child: Row(
                    children: const [
                      Icon(Icons.offline_pin, color: Colors.orange),
                      SizedBox(width: 12),
                      Expanded(
                        child: Text(
                          'Offline First Mode: Scans are saved locally in SQLite and automatically synchronized when online.',
                          style: TextStyle(fontSize: 13, color: Colors.black87),
                        ),
                      ),
                    ],
                  ),
                ),
              ),
              const SizedBox(height: 16),
              Row(
                children: [
                  Expanded(
                    child: TextFormField(
                      controller: _qrCodeController,
                      decoration: const InputDecoration(
                        labelText: 'QR Code Payload *',
                        border: OutlineInputBorder(),
                        prefixIcon: Icon(Icons.qr_code_2),
                      ),
                      validator: (val) => val == null || val.trim().isEmpty ? 'QR Code is required' : null,
                    ),
                  ),
                  const SizedBox(width: 8),
                  IconButton(
                    onPressed: _scanQRCode,
                    icon: const Icon(Icons.qr_code_scanner),
                    style: IconButton.styleFrom(
                      backgroundColor: Colors.orange.shade800,
                      foregroundColor: Colors.white,
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 16),
              TextFormField(
                controller: _assetIdController,
                decoration: const InputDecoration(
                  labelText: 'Asset UUID *',
                  border: OutlineInputBorder(),
                  prefixIcon: Icon(Icons.hardware),
                ),
                validator: (val) => val == null || val.trim().isEmpty ? 'Asset ID is required' : null,
              ),
              const SizedBox(height: 16),
              TextFormField(
                controller: _ticketIdController,
                decoration: const InputDecoration(
                  labelText: 'Maintenance Ticket UUID (Optional)',
                  border: OutlineInputBorder(),
                  prefixIcon: Icon(Icons.confirmation_number),
                ),
              ),
              const SizedBox(height: 16),
              TextFormField(
                controller: _actionTakenController,
                maxLines: 3,
                decoration: const InputDecoration(
                  labelText: 'Action Taken *',
                  border: OutlineInputBorder(),
                  prefixIcon: Icon(Icons.build),
                ),
                validator: (val) => val == null || val.trim().isEmpty ? 'Action taken is required' : null,
              ),
              const SizedBox(height: 16),
              TextFormField(
                controller: _notesController,
                maxLines: 2,
                decoration: const InputDecoration(
                  labelText: 'Technician Notes',
                  border: OutlineInputBorder(),
                  prefixIcon: Icon(Icons.note),
                ),
              ),
              const SizedBox(height: 24),
              ElevatedButton.icon(
                onPressed: _isSaving ? null : () => _saveLog(syncImmediately: false),
                style: ElevatedButton.styleFrom(
                  backgroundColor: Colors.grey.shade800,
                  foregroundColor: Colors.white,
                  padding: const EdgeInsets.symmetric(vertical: 14),
                ),
                icon: const Icon(Icons.save),
                label: const Text('Save Offline in Local SQLite'),
              ),
              const SizedBox(height: 12),
              ElevatedButton.icon(
                onPressed: _isSaving ? null : () => _saveLog(syncImmediately: true),
                style: ElevatedButton.styleFrom(
                  backgroundColor: Colors.orange.shade800,
                  foregroundColor: Colors.white,
                  padding: const EdgeInsets.symmetric(vertical: 14),
                ),
                icon: const Icon(Icons.cloud_upload),
                label: const Text('Save & Sync Now'),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
