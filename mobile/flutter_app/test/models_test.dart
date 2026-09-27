import 'package:flutter_test/flutter_test.dart';
import 'package:oci_field_maintenance/models/maintenance_log.dart';
import 'package:oci_field_maintenance/models/hardware_asset.dart';
import 'package:oci_field_maintenance/models/ota_job.dart';

void main() {
  group('MaintenanceLog Model Tests', () {
    test('MaintenanceLog toMap and fromMap conversion', () {
      final now = DateTime.utc(2026, 3, 30, 12, 0, 0);
      final log = MaintenanceLog(
        id: 'log-123',
        assetId: 'asset-456',
        ticketId: 'ticket-789',
        technicianId: 'emp-human-bob',
        qrCodeScanned: 'QR-CAM-01',
        actionTaken: 'Clean lens',
        notes: 'Checked power cables',
        synced: false,
        createdAt: now,
      );

      final map = log.toMap();
      expect(map['id'], 'log-123');
      expect(map['asset_id'], 'asset-456');
      expect(map['synced'], 0);

      final reconstructed = MaintenanceLog.fromMap(map);
      expect(reconstructed.id, log.id);
      expect(reconstructed.assetId, log.assetId);
      expect(reconstructed.synced, false);
      expect(reconstructed.qrCodeScanned, log.qrCodeScanned);
    });

    test('MaintenanceLog toJson conversion', () {
      final now = DateTime.utc(2026, 3, 30, 12, 0, 0);
      final log = MaintenanceLog(
        id: 'log-123',
        assetId: 'asset-456',
        qrCodeScanned: 'QR-CAM-01',
        actionTaken: 'Replaced mount',
        createdAt: now,
      );

      final json = log.toJson();
      expect(json['asset_id'], 'asset-456');
      expect(json['qr_code_scanned'], 'QR-CAM-01');
      expect(json['technician_id'], 'emp-human-bob');
    });
  });

  group('HardwareAsset & OTAJob Model Tests', () {
    test('HardwareAsset.fromJson', () {
      final json = {
        'asset_id': 'asset-1',
        'serial_number': 'SER-001',
        'asset_type': 'edge_camera',
        'model': '4K-Cam',
        'status': 'active',
      };
      final asset = HardwareAsset.fromJson(json);
      expect(asset.assetId, 'asset-1');
      expect(asset.assetType, 'edge_camera');
    });

    test('OTAJob.fromJson', () {
      final json = {
        'job_id': 'job-1',
        'asset_id': 'asset-1',
        'release_id': 'rel-1',
        'status': 'completed',
        'scheduled_at': '2026-03-30T10:00:00Z',
      };
      final job = OTAJob.fromJson(json);
      expect(job.jobId, 'job-1');
      expect(job.status, 'completed');
    });
  });
}
