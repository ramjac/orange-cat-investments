import 'dart:convert';
import 'package:http/http.dart' as http;
import '../models/maintenance_log.dart';
import '../models/hardware_asset.dart';
import '../models/ota_job.dart';

class ApiService {
  final String baseUrl;
  final http.Client client;

  ApiService({
    this.baseUrl = 'https://employee.oci.local/api/v1',
    http.Client? client,
  }) : client = client ?? http.Client();

  Future<bool> syncMaintenanceLogs(List<MaintenanceLog> logs) async {
    if (logs.isEmpty) return true;
    try {
      final body = jsonEncode(logs.map((l) => l.toJson()).toList());
      final response = await client.post(
        Uri.parse('$baseUrl/facilities/sync/maintenance-logs'),
        headers: {
          'Content-Type': 'application/json',
          'X-CSRF-Token': 'mobile-field-app-csrf',
        },
        body: body,
      );
      return response.statusCode == 200 || response.statusCode == 201;
    } catch (_) {
      return false;
    }
  }

  Future<List<HardwareAsset>> fetchHardwareAssets() async {
    try {
      final response = await client.get(
        Uri.parse('$baseUrl/facilities/assets'),
      );
      if (response.statusCode == 200) {
        final data = jsonDecode(response.body);
        final List items = data['items'] ?? [];
        return items.map((item) => HardwareAsset.fromJson(item)).toList();
      }
    } catch (_) {}
    return [];
  }

  Future<List<OTAJob>> fetchOTAJobs(String assetId) async {
    try {
      final response = await client.get(
        Uri.parse('$baseUrl/facilities/firmware/ota-jobs?asset_id=$assetId'),
      );
      if (response.statusCode == 200) {
        final List data = jsonDecode(response.body);
        return data.map((item) => OTAJob.fromJson(item)).toList();
      }
    } catch (_) {}
    return [];
  }
}
