class HardwareAsset {
  final String assetId;
  final String serialNumber;
  final String assetType;
  final String model;
  final String status;
  final String? zoneId;

  HardwareAsset({
    required this.assetId,
    required this.serialNumber,
    required this.assetType,
    required this.model,
    required this.status,
    this.zoneId,
  });

  factory HardwareAsset.fromJson(Map<String, dynamic> json) {
    return HardwareAsset(
      assetId: json['asset_id'] as String,
      serialNumber: json['serial_number'] as String,
      assetType: json['asset_type'] as String,
      model: json['model'] as String,
      status: json['status'] as String,
      zoneId: json['zone_id'] as String?,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'asset_id': assetId,
      'serial_number': serialNumber,
      'asset_type': assetType,
      'model': model,
      'status': status,
      'zone_id': zoneId,
    };
  }
}
