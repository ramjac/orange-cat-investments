class OTAJob {
  final String jobId;
  final String assetId;
  final String releaseId;
  final String status;
  final String? errorMessage;
  final DateTime scheduledAt;
  final DateTime? completedAt;

  OTAJob({
    required this.jobId,
    required this.assetId,
    required this.releaseId,
    required this.status,
    this.errorMessage,
    required this.scheduledAt,
    this.completedAt,
  });

  factory OTAJob.fromJson(Map<String, dynamic> json) {
    return OTAJob(
      jobId: json['job_id'] as String,
      assetId: json['asset_id'] as String,
      releaseId: json['release_id'] as String,
      status: json['status'] as String,
      errorMessage: json['error_message'] as String?,
      scheduledAt: DateTime.parse(json['scheduled_at'] as String),
      completedAt: json['completed_at'] != null ? DateTime.parse(json['completed_at'] as String) : null,
    );
  }
}
