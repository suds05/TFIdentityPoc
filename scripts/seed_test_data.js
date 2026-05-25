// //////////////////////////////////////////////////////////
// //
// // Copyright 2026 Sudhakar Narayanamurthy. All rights reserved.
// // Licensed under the Apache License, Version 2.0 (the "License")
// //
// // Idempotent mongosh seed for GlobalDB and both storage tier databases.
// //

function upsertOne(dbName, collection, filter, doc) {
  const result = db
    .getSiblingDB(dbName)
    .getCollection(collection)
    .replaceOne(filter, doc, { upsert: true });
  if (!result.acknowledged) {
    throw new Error(`upsert failed: ${dbName}.${collection}`);
  }
}

// GlobalDB
upsertOne("global", "user_team_memberships", { _id: "usr_sudhakan" }, {
  _id: "usr_sudhakan",
  email: "sudhakan@gmail.com",
  teamIds: ["engineering", "marketing"],
});

const routing = [
  { _id: "engineering", storageTierId: 1 },
  { _id: "qa", storageTierId: 1 },
  { _id: "devops", storageTierId: 1 },
  { _id: "marketing", storageTierId: 2 },
  { _id: "sales", storageTierId: 2 },
];
for (const doc of routing) {
  upsertOne("global", "team_storage_routing", { _id: doc._id }, doc);
}

// Storage tier 1
const tier1Teams = [
  {
    _id: "engineering",
    folders: [
      { folderId: "code", name: "Code" },
      { folderId: "specs", name: "Specs" },
    ],
  },
  {
    _id: "qa",
    folders: [
      { folderId: "tests", name: "Tests" },
      { folderId: "results", name: "Results" },
    ],
  },
  {
    _id: "devops",
    folders: [
      { folderId: "infrastructure", name: "Infrastructure" },
      { folderId: "monitoring", name: "Monitoring" },
    ],
  },
];
for (const doc of tier1Teams) {
  upsertOne("storage_tier_1", "teams", { _id: doc._id }, doc);
}

// Storage tier 2
const tier2Teams = [
  {
    _id: "marketing",
    folders: [
      { folderId: "campaigns", name: "Campaigns" },
      { folderId: "creative", name: "Creative" },
    ],
  },
  {
    _id: "sales",
    folders: [
      { folderId: "proposals", name: "Proposals" },
      { folderId: "contracts", name: "Contracts" },
    ],
  },
];
for (const doc of tier2Teams) {
  upsertOne("storage_tier_2", "teams", { _id: doc._id }, doc);
}

print("seed complete: global (2 collections), storage_tier_1, storage_tier_2");
