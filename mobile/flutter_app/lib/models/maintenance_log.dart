class MaintenanceLog {
  final String id;
  final String? ticketId;
  final String assetId;
  final String? technicianId;
  final String qrCodeScanned;
  final String actionTaken;
  final String? notes;
  final bool synced;
  final DateTime createdAt;
  final DateTime? syncedAt;

  MaintenanceLog({
    required this.id,
    this.ticketId,
    required this.assetId,
    this.technicianId,
    required this.qrCodeScanned,
    required this.actionTaken,
    this.notes,
    this.synced = false,
    required this.createdAt,
    this.syncedAt,
  });

  Map<String, dynamic> toMap() {
    return {
      'id': id,
      'ticket_id': ticketId,
      'asset_id': assetId,
      'technician_id': technicianId,
      'qr_code_scanned': qrCodeScanned,
      'action_taken': actionTaken,
      'notes': notes,
      'synced': synced ? 1 : 0,
      'created_at': createdAt.toIso8601String(),
      'synced_at': syncedAt?.toIso8601String(),
    };
  }

  factory MaintenanceLog.fromMap(Map<String, dynamic> map) {
    return MaintenanceLog(
      id: map['id'] as String,
      ticketId: map['ticket_id'] as String?,
      assetId: map['asset_id'] as String,
      technicianId: map['technician_id'] as String?,
      qrCodeScanned: map['qr_code_scanned'] as String,
      actionTaken: map['action_taken'] as String,
      notes: map['notes'] as String?,
      synced: (map['synced'] as int?) == 1,
      createdAt: DateTime.parse(map['created_at'] as String),
      syncedAt: map['synced_at'] != null ? DateTime.parse(map['synced_at'] as String) : null,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'ticket_id': ticketId,
      'asset_id': assetId,
      'technician_id': technicianId ?? 'emp-human-bob',
      'qr_code_scanned': qrCodeScanned,
      'action_taken': actionTaken,
      'notes': notes,
      'created_at': createdAt.toIso8601String(),
    };
  }

  MaintenanceLog copyWith({
    bool? synced,
    DateTime? syncedAt,
  }) {
    return MaintenanceLog(
      id: id,
      ticketId: ticketId,
      assetId: assetId,
      technicianId: technicianId,
      qrCodeScanned: qrCodeScanned,
      actionTaken: actionTaken,
      notes: notes,
      synced: synced ?? this.synced,
      createdAt: createdAt,
      syncedAt: syncedAt ?? this.syncedAt,
    );
  }
}
