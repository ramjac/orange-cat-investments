import 'package:flutter_test/flutter_test.dart';
import 'package:oci_field_maintenance/main.dart';
import 'package:oci_field_maintenance/services/database_helper.dart';
import 'package:oci_field_maintenance/services/api_service.dart';
import 'package:oci_field_maintenance/services/sync_engine.dart';
import 'package:oci_field_maintenance/models/maintenance_log.dart';

class DummyApiService extends ApiService {
  @override
  Future<bool> syncMaintenanceLogs(logs) async => true;
}

class MockDatabaseHelper implements DatabaseHelper {
  final List<MaintenanceLog> _logs = [];

  @override
  Future<int> insertLog(MaintenanceLog log) async {
    _logs.add(log);
    return 1;
  }

  @override
  Future<List<MaintenanceLog>> getUnsyncedLogs() async {
    return _logs.where((l) => !l.synced).toList();
  }

  @override
  Future<List<MaintenanceLog>> getAllLogs() async {
    return List.from(_logs);
  }

  @override
  Future<int> markAsSynced(List<String> logIds, DateTime syncedAt) async {
    return logIds.length;
  }

  @override
  Future<void> close() async {}

  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

void main() {
  testWidgets('Flutter UI Dashboard Renders Persona and Action Cards', (WidgetTester tester) async {
    final dbHelper = MockDatabaseHelper();
    final apiService = DummyApiService();
    final syncEngine = SyncEngine(dbHelper: dbHelper, apiService: apiService);

    await tester.pumpWidget(OCIFieldMaintenanceApp(
      syncEngine: syncEngine,
      apiService: apiService,
    ));

    await tester.pumpAndSettle();

    expect(find.text('Bob Builder'), findsOneWidget);
    expect(find.text('Lead Facilities & Edge Telemetry Engineer'), findsOneWidget);
    expect(find.text('Scan QR Code'), findsOneWidget);
    expect(find.text('Log Maintenance'), findsOneWidget);
    expect(find.text('Offline Queue'), findsOneWidget);
    expect(find.text('Firmware OTA'), findsOneWidget);
  });
}
