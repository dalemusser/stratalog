// Migration script to rename dbtimestamp to serverTimestamp
// Run: mongosh stratalog scripts/rename_dbtimestamp.js

print("=== Renaming dbtimestamp to serverTimestamp ===");

const coll = db.logdata;

// Count documents with dbtimestamp field
const countBefore = coll.countDocuments({ dbtimestamp: { $exists: true } });
print("Documents with dbtimestamp: " + countBefore);

if (countBefore === 0) {
    print("No documents to migrate.");
} else {
    // Rename field from dbtimestamp to serverTimestamp
    const result = coll.updateMany(
        { dbtimestamp: { $exists: true } },
        { $rename: { "dbtimestamp": "serverTimestamp" } }
    );

    print("Modified: " + result.modifiedCount + " documents");
}

// Verify
const countAfter = coll.countDocuments({ serverTimestamp: { $exists: true } });
print("Documents with serverTimestamp: " + countAfter);

print("=== Migration Complete ===");
