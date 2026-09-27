import 'package:flutter/material.dart';
import 'services/database_helper.dart';
import 'services/api_service.dart';
import 'services/sync_engine.dart';
import 'views/home_screen.dart';

void main() async {
  WidgetsFlutterBinding.ensureInitialized();
  final dbHelper = DatabaseHelper.instance;
  final apiService = ApiService();
  final syncEngine = SyncEngine(dbHelper: dbHelper, apiService: apiService);

  runApp(OCIFieldMaintenanceApp(
    syncEngine: syncEngine,
    apiService: apiService,
  ));
}

class OCIFieldMaintenanceApp extends StatelessWidget {
  final SyncEngine syncEngine;
  final ApiService apiService;

  const OCIFieldMaintenanceApp({
    super.key,
    required this.syncEngine,
    required this.apiService,
  });

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'OCI Field Maintenance',
      debugShowCheckedModeBanner: false,
      theme: ThemeData(
        useMaterial3: true,
        colorScheme: ColorScheme.fromSeed(
          seedColor: Colors.deepOrange,
          primary: Colors.orange.shade800,
        ),
      ),
      home: HomeScreen(
        syncEngine: syncEngine,
        apiService: apiService,
      ),
    );
  }
}
