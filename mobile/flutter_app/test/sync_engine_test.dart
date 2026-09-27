import 'package:flutter_test/flutter_test.dart';
import 'package:sqflite_common_ffi/sqflite_ffi.dart';
import 'package:oci_field_maintenance/services/database_helper.dart';
import 'package:oci_field_maintenance/services/api_service.dart';
import 'package:oci_field_maintenance/services/sync_engine.dart';
import 'package:oci_field_maintenance/models/maintenance_log.dart';

class MockSuccessApiService extends ApiService {
  int syncCalls = 0;

  @override
  Future<bool> syncMaintenanceLogs(List<MaintenanceLog> logs) async {
    syncCalls++;
    return true;
  }
}

class MockFailureApiService extends ApiService {
  @override
  Future<bool> syncMaintenanceLogs(List<MaintenanceLog> logs) async {
    return false;
  }
}

void main() {
  setUpAll(() {
    sqfliteFfiInit();
    databaseFactory = databaseFactoryFfi;
  });

  group('SyncEngine Tests', () {
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

    test('SyncEngine syncs unsynced logs upon returning online', () async {
      final mockApi = MockSuccessApiService();
      final syncEngine = SyncEngine(dbHelper: dbHelper, apiService: mockApi);

      final log = MaintenanceLog(
        id: 'sync-log-1',
        assetId: 'asset-1',
        qrCodeScanned: 'QR-01',
        actionTaken: 'Repaired cable connection',
        createdAt: DateTime.now().toUtc(),
      );

      await syncEngine.saveLogLocally(log);

      final result = await syncEngine.syncPendingLogs();
      expect(result.isSuccess, true);
      expect(result.syncedCount, 1);
      expect(result.remainingPending, 0);
      expect(mockApi.syncCalls, 1);

      final unsynced = await dbHelper.getUnsyncedLogs();
      expect(unsynced.isEmpty, true);
    });

    test('SyncEngine retains logs in SQLite when offline or sync fails', () async {
      final mockApi = MockFailureApiService();
      final syncEngine = SyncEngine(dbHelper: dbHelper, apiService: mockApi);

      final log = MaintenanceLog(
        id: 'sync-log-offline',
        assetId: 'asset-offline',
        qrCodeScanned: 'QR-OFFLINE',
        actionTaken: 'Logged while in subterranean perch zone',
        createdAt: DateTime.now().toUtc(),
      );

      await syncEngine.saveLogLocally(log);

      final result = await syncEngine.syncPendingLogs();
      expect(result.isSuccess, false);
      expect(result.syncedCount, 0);
      expect(result.remainingPending, 1);

      final unsynced = await dbHelper.getUnsyncedLogs();
      expect(unsynced.length, 1);
      expect(unsynced.first.id, 'sync-log-offline');
    });
  });
}
