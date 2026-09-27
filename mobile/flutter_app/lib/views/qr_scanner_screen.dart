import 'package:flutter/material.dart';

class QRScannerScreen extends StatefulWidget {
  const QRScannerScreen({super.key});

  @override
  State<QRScannerScreen> createState() => _QRScannerScreenState();
}

class _QRScannerScreenState extends State<QRScannerScreen> {
  final TextEditingController _qrInputController = TextEditingController();

  void _onQRScanned(String value) {
    if (value.trim().isEmpty) return;
    Navigator.of(context).pop(value.trim());
  }

  @override
  void dispose() {
    _qrInputController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('QR Code Asset Scanner'),
        backgroundColor: Colors.orange.shade800,
        foregroundColor: Colors.white,
      ),
      body: Padding(
        padding: const EdgeInsets.all(24.0),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Container(
              height: 200,
              decoration: BoxDecoration(
                color: Colors.grey.shade900,
                borderRadius: BorderRadius.circular(16),
                border: Border.all(color: Colors.orange, width: 2),
              ),
              child: Column(
                mainAxisAlignment: MainAxisAlignment.center,
                children: const [
                  Icon(Icons.qr_code_scanner, size: 80, color: Colors.orangeAccent),
                  SizedBox(height: 12),
                  Text(
                    'Simulated Camera View\nAlign QR Code within frame',
                    textAlign: TextAlign.center,
                    style: TextStyle(color: Colors.white70, fontSize: 14),
                  ),
                ],
              ),
            ),
            const SizedBox(height: 24),
            const Text(
              'Or enter QR code payload manually:',
              style: TextStyle(fontWeight: FontWeight.bold),
            ),
            const SizedBox(height: 8),
            TextField(
              controller: _qrInputController,
              decoration: const InputDecoration(
                border: OutlineInputBorder(),
                hintText: 'e.g., QR-CAM-ORANGE-01 / asset-uuid-001',
                prefixIcon: Icon(Icons.qr_code),
              ),
            ),
            const SizedBox(height: 16),
            ElevatedButton.icon(
              onPressed: () => _onQRScanned(_qrInputController.text),
              style: ElevatedButton.styleFrom(
                backgroundColor: Colors.orange.shade800,
                foregroundColor: Colors.white,
                padding: const EdgeInsets.symmetric(vertical: 14),
              ),
              icon: const Icon(Icons.check),
              label: const Text('Confirm Scanned QR Code'),
            ),
            const SizedBox(height: 12),
            Wrap(
              spacing: 8,
              children: [
                ActionChip(
                  label: const Text('Sample Camera 01'),
                  onPressed: () => _onQRScanned('QR-CAM-ORANGE-01'),
                ),
                ActionChip(
                  label: const Text('Sample Perch 02'),
                  onPressed: () => _onQRScanned('QR-PERCH-ALPHA-02'),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}
