import 'package:flutter_test/flutter_test.dart';
import 'package:sqflite_common_ffi/sqflite_ffi.dart';
import 'package:oci_field_maintenance/services/database_helper.dart';

void main() {
  setUpAll(() {
    sqfliteFfiInit();
    databaseFactory = databaseFactoryFfi;
  });

  group('DatabaseHelper Performance Benchmark', () {
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

    Future<void> seedDatabase(int count) async {
      final now = DateTime.utc(2026, 3, 30, 14, 0, 0);
      final batch = db.batch();
      for (int i = 0; i < count; i++) {
        batch.insert('maintenance_logs', {
          'id': 'log-$i',
          'asset_id': 'asset-$i',
          'qr_code_scanned': 'QR-$i',
          'action_taken': 'Action $i',
          'synced': 0,
          'created_at': now.toIso8601String(),
        });
      }
      await batch.commit(noResult: true);
    }

    test('Benchmark markAsSynced with 500 records', () async {
      await seedDatabase(500);
      final syncedAt = DateTime.now().toUtc();
      final logIds = List.generate(500, (i) => 'log-$i');

      final sw = Stopwatch()..start();
      final count = await dbHelper.markAsSynced(logIds, syncedAt);
      sw.stop();

      print('markAsSynced with 500 records took: ${sw.elapsedMilliseconds} ms (updated: $count)');
      expect(count, 500);

      final unsynced = await dbHelper.getUnsyncedLogs();
      expect(unsynced.length, 0);
    });

    test('Benchmark markAsSynced with 2000 records', () async {
      await seedDatabase(2000);
      final syncedAt = DateTime.now().toUtc();
      final logIds = List.generate(2000, (i) => 'log-$i');

      final sw = Stopwatch()..start();
      final count = await dbHelper.markAsSynced(logIds, syncedAt);
      sw.stop();

      print('markAsSynced with 2000 records took: ${sw.elapsedMilliseconds} ms (updated: $count)');
      expect(count, 2000);

      final unsynced = await dbHelper.getUnsyncedLogs();
      expect(unsynced.length, 0);
    });
  });
}
