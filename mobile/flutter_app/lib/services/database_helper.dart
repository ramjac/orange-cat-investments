import 'package:sqflite/sqflite.dart';
import 'package:path/path.dart' as p;
import '../models/maintenance_log.dart';

class DatabaseHelper {
  static final DatabaseHelper instance = DatabaseHelper._init();
  static Database? _database;

  DatabaseHelper._init();

  DatabaseHelper.withDb(Database db) {
    _database = db;
  }

  Future<Database> get database async {
    if (_database != null) return _database!;
    _database = await _initDB('oci_maintenance.db');
    return _database!;
  }

  Future<Database> _initDB(String filePath) async {
    final dbPath = await getDatabasesPath();
    final path = p.join(dbPath, filePath);

    return await openDatabase(
      path,
      version: 1,
      onCreate: _createDB,
    );
  }

  Future<void> _createDB(Database db, int version) async {
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
  }

  Future<int> insertLog(MaintenanceLog log) async {
    final db = await database;
    return await db.insert(
      'maintenance_logs',
      log.toMap(),
      conflictAlgorithm: ConflictAlgorithm.replace,
    );
  }

  Future<List<MaintenanceLog>> getUnsyncedLogs() async {
    final db = await database;
    final maps = await db.query(
      'maintenance_logs',
      where: 'synced = ?',
      whereArgs: [0],
      orderBy: 'created_at ASC',
    );
    return maps.map((map) => MaintenanceLog.fromMap(map)).toList();
  }

  Future<List<MaintenanceLog>> getAllLogs() async {
    final db = await database;
    final maps = await db.query(
      'maintenance_logs',
      orderBy: 'created_at DESC',
    );
    return maps.map((map) => MaintenanceLog.fromMap(map)).toList();
  }

  Future<int> markAsSynced(List<String> logIds, DateTime syncedAt) async {
    if (logIds.isEmpty) return 0;
    final db = await database;
    int count = 0;
    for (final id in logIds) {
      count += await db.update(
        'maintenance_logs',
        {
          'synced': 1,
          'synced_at': syncedAt.toIso8601String(),
        },
        where: 'id = ?',
        whereArgs: [id],
      );
    }
    return count;
  }

  Future<void> close() async {
    final db = _database;
    if (db != null) {
      await db.close();
      _database = null;
    }
  }
}
