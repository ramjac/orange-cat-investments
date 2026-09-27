import 'package:flutter_test/flutter_test.dart';
import 'package:sqflite_common_ffi/sqflite_ffi.dart';
import 'package:oci_field_maintenance/services/database_helper.dart';
import 'package:oci_field_maintenance/models/maintenance_log.dart';

void main() {
  setUpAll(() {
    sqfliteFfiInit();
    databaseFactory = databaseFactoryFfi;
  });

  group('DatabaseHelper SQLite Offline Persistence Tests', () {
    late Database db;
    late DatabaseHelper dbHelper;

    setUp(() async {
      db = await openDatabase(
        inMemoryDatabasePath,
        version: 1,
        onCreate: (db, version) async {
          await db.execute('''
            CREATE TABLE maintenance_logs (
              id TEXT PRIMARY KEY,
              ticket_id TEXT,
              asset_id TEXT NOT NULL,
              technician_id TEXT,
              qr_code_scanned TEXT NOT NULL,
              action_taken TEXT NOT NULL,
              notes TEXT,
              synced INTEGER NOT NULL DEFAULT 0,
              created_at TEXT NOT NULL,
              synced_at TEXT
            )
          ''');
        },
      );
      dbHelper = DatabaseHelper.withDb(db);
    });

    tearDown(() async {
      await dbHelper.close();
    });

    test('Insert log and retrieve unsynced logs', () async {
      final now = DateTime.utc(2026, 3, 30, 14, 0, 0);
      final log = MaintenanceLog(
        id: 'log-sqflite-1',
        assetId: 'asset-sqflite-1',
        qrCodeScanned: 'QR-TEST-01',
        actionTaken: 'Calibrated orientation sensor',
        createdAt: now,
      );

      await dbHelper.insertLog(log);

      final unsynced = await dbHelper.getUnsyncedLogs();
      expect(unsynced.length, 1);
      expect(unsynced.first.id, 'log-sqflite-1');
      expect(unsynced.first.synced, false);
    });

    test('Mark logs as synced updates SQLite records', () async {
      final now = DateTime.utc(2026, 3, 30, 14, 0, 0);
      final log = MaintenanceLog(
        id: 'log-sqflite-2',
        assetId: 'asset-sqflite-2',
        qrCodeScanned: 'QR-TEST-02',
        actionTaken: 'Replaced battery',
        createdAt: now,
      );

      await dbHelper.insertLog(log);
      final syncTime = DateTime.utc(2026, 3, 30, 14, 05, 0);
      await dbHelper.markAsSynced(['log-sqflite-2'], syncTime);

      final unsynced = await dbHelper.getUnsyncedLogs();
      expect(unsynced.length, 0);

      final allLogs = await dbHelper.getAllLogs();
      expect(allLogs.length, 1);
      expect(allLogs.first.synced, true);
      expect(allLogs.first.syncedAt, syncTime);
    });
  });
}
