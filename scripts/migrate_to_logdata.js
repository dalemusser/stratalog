// Migration script to consolidate log collections into unified logdata collection
// Run: mongosh stratalog scripts/migrate_to_logdata.js

const TARGET = "logdata";

// Get all collection names
const collections = db.getCollectionNames();

// Find source collections (logs_* prefix - current stratalog format)
const logCollections = collections.filter(n => n.startsWith("logs_"));

// Find raw game collections (original strata_log format - no prefix)
// Exclude known system collections
const systemCollections = [
    "logdata", "api_stats", "users", "sessions", "settings", "announcements"
];
const rawGameCollections = collections.filter(n =>
    !n.startsWith("logs_") &&
    !n.startsWith("system.") &&
    !systemCollections.includes(n)
);

print("=== StrataLog Migration to logdata ===");
print("Target collection: " + TARGET);
print("logs_* collections found: " + logCollections.length);
print("Raw game collections found: " + rawGameCollections.length);
print("");

// Migrate logs_* collections (current stratalog format - may have data field)
logCollections.forEach(function(coll) {
    const game = coll.replace("logs_", "");
    migrateCollection(coll, game, true);  // flatten data field
});

// Migrate raw game collections (original strata_log format - already flat)
rawGameCollections.forEach(function(coll) {
    migrateCollection(coll, coll, false);  // already flat
});

function migrateCollection(sourceColl, gameName, flattenData) {
    const cursor = db[sourceColl].find();
    let migrated = 0, skipped = 0;

    while (cursor.hasNext()) {
        const doc = cursor.next();
        const newDoc = { _id: doc._id, game: gameName };

        // Copy all fields except _id and data
        Object.keys(doc).forEach(k => {
            if (k !== "_id" && k !== "data") {
                newDoc[k] = doc[k];
            }
        });

        // Flatten data field if present (stratalog format)
        if (flattenData && doc.data) {
            Object.keys(doc.data).forEach(k => {
                // Only copy if not already present at root level
                if (!(k in newDoc)) {
                    newDoc[k] = doc.data[k];
                }
            });
        }

        try {
            db[TARGET].insertOne(newDoc);
            migrated++;
        } catch (e) {
            if (e.code === 11000) {
                skipped++;  // duplicate _id, already migrated
            } else {
                print("Error migrating doc " + doc._id + ": " + e.message);
            }
        }
    }
    print(sourceColl + ": migrated=" + migrated + ", skipped=" + skipped);
}

print("");
print("=== Creating Indexes ===");

// Create indexes for efficient querying
db[TARGET].createIndex(
    {game: 1, serverTimestamp: -1},
    {name: "idx_game_serverTimestamp", background: true}
);
print("Created index: idx_game_serverTimestamp");

db[TARGET].createIndex(
    {game: 1, playerId: 1},
    {name: "idx_game_playerId", background: true}
);
print("Created index: idx_game_playerId");

db[TARGET].createIndex(
    {game: 1, eventType: 1},
    {name: "idx_game_eventType", background: true}
);
print("Created index: idx_game_eventType");

print("");
print("=== Migration Complete ===");
print("Total documents in logdata: " + db[TARGET].countDocuments());
print("");
print("To verify, run:");
print("  db.logdata.distinct('game')");
print("  db.logdata.findOne({game: '<game_name>'})");
