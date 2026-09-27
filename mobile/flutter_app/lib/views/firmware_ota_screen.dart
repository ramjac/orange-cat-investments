import 'package:flutter/material.dart';
import '../models/ota_job.dart';
import '../services/api_service.dart';

class FirmwareOTAScreen extends StatefulWidget {
  final ApiService apiService;

  const FirmwareOTAScreen({
    super.key,
    required this.apiService,
  });

  @override
  State<FirmwareOTAScreen> createState() => _FirmwareOTAScreenState();
}

class _FirmwareOTAScreenState extends State<FirmwareOTAScreen> {
  final TextEditingController _assetIdController = TextEditingController(text: 'asset-uuid-001');
  List<OTAJob> _jobs = [];
  bool _isLoading = false;

  @override
  void initState() {
    super.initState();
    _fetchJobs();
  }

  @override
  void dispose() {
    _assetIdController.dispose();
    super.dispose();
  }

  Future<void> _fetchJobs() async {
    final assetId = _assetIdController.text.trim();
    if (assetId.isEmpty) return;

    setState(() => _isLoading = true);
    final jobs = await widget.apiService.fetchOTAJobs(assetId);
    setState(() {
      _jobs = jobs;
      _isLoading = false;
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Edge Firmware OTA Pipeline'),
        backgroundColor: Colors.orange.shade800,
        foregroundColor: Colors.white,
      ),
      body: Padding(
        padding: const EdgeInsets.all(16.0),
        child: Column(
          children: [
            Row(
              children: [
                Expanded(
                  child: TextField(
                    controller: _assetIdController,
                    decoration: const InputDecoration(
                      labelText: 'Hardware Asset ID',
                      border: OutlineInputBorder(),
                      prefixIcon: Icon(Icons.memory),
                    ),
                  ),
                ),
                const SizedBox(width: 8),
                ElevatedButton.icon(
                  onPressed: _isLoading ? null : _fetchJobs,
                  style: ElevatedButton.styleFrom(
                    backgroundColor: Colors.orange.shade800,
                    foregroundColor: Colors.white,
                    padding: const EdgeInsets.symmetric(vertical: 16, horizontal: 16),
                  ),
                  icon: const Icon(Icons.search),
                  label: const Text('Check Status'),
                ),
              ],
            ),
            const SizedBox(height: 16),
            Expanded(
              child: _isLoading
                  ? const Center(child: CircularProgressIndicator())
                  : _jobs.isEmpty
                      ? const Center(
                          child: Text('No OTA Firmware Update jobs found for this asset.'),
                        )
                      : ListView.builder(
                          itemCount: _jobs.length,
                          itemBuilder: (context, index) {
                            final job = _jobs[index];
                            final isCompleted = job.status == 'completed';
                            return Card(
                              child: ListTile(
                                leading: Icon(
                                  isCompleted ? Icons.check_circle : Icons.system_update,
                                  color: isCompleted ? Colors.green : Colors.orange,
                                  size: 32,
                                ),
                                title: Text('OTA Job: ${job.jobId}'),
                                subtitle: Text(
                                  'Status: ${job.status.toUpperCase()}\n'
                                  'Scheduled: ${job.scheduledAt}\n'
                                  'Completed: ${job.completedAt ?? 'N/A'}',
                                ),
                              ),
                            );
                          },
                        ),
            ),
          ],
        ),
      ),
    );
  }
}
