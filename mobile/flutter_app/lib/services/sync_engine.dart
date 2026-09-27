import '../models/maintenance_log.dart';
import 'database_helper.dart';
import 'api_service.dart';

class SyncResult {
  final int syncedCount;
  final int remainingPending;
  final bool isSuccess;
  final String? errorMessage;

  SyncResult({
    required this.syncedCount,
    required this.remainingPending,
    required this.isSuccess,
    this.errorMessage,
  });
}

class SyncEngine {
  final DatabaseHelper dbHelper;
  final ApiService apiService;

  SyncEngine({
    DatabaseHelper? dbHelper,
    ApiService? apiService,
  })  : dbHelper = dbHelper ?? DatabaseHelper.instance,
        apiService = apiService ?? ApiService();

  Future<SyncResult> syncPendingLogs() async {
    try {
      final unsynced = await dbHelper.getUnsyncedLogs();
      if (unsynced.isEmpty) {
        return SyncResult(
          syncedCount: 0,
          remainingPending: 0,
          isSuccess: true,
        );
      }

      final success = await apiService.syncMaintenanceLogs(unsynced);
      if (success) {
        final now = DateTime.now().toUtc();
        final ids = unsynced.map((l) => l.id).toList();
        await dbHelper.markAsSynced(ids, now);

        final remaining = await dbHelper.getUnsyncedLogs();
        return SyncResult(
          syncedCount: ids.length,
          remainingPending: remaining.length,
          isSuccess: true,
        );
      } else {
        return SyncResult(
          syncedCount: 0,
          remainingPending: unsynced.length,
          isSuccess: false,
          errorMessage: 'Server synchronization failed or device is offline.',
        );
      }
    } catch (e) {
      final remaining = await dbHelper.getUnsyncedLogs();
      return SyncResult(
        syncedCount: 0,
        remainingPending: remaining.length,
        isSuccess: false,
        errorMessage: e.toString(),
      );
    }
  }

  Future<void> saveLogLocally(MaintenanceLog log) async {
    await dbHelper.insertLog(log);
  }
}
