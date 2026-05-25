// Verify POC seed data document counts.
const expected = {
  global_user_team_memberships: 1,
  global_team_storage_routing: 5,
  storage_tier_1_teams: 3,
  storage_tier_2_teams: 2,
};

const actual = {
  global_user_team_memberships: db
    .getSiblingDB("global")
    .user_team_memberships.countDocuments(),
  global_team_storage_routing: db
    .getSiblingDB("global")
    .team_storage_routing.countDocuments(),
  storage_tier_1_teams: db.getSiblingDB("storage_tier_1").teams.countDocuments(),
  storage_tier_2_teams: db.getSiblingDB("storage_tier_2").teams.countDocuments(),
};

printjson(actual);

for (const [key, want] of Object.entries(expected)) {
  const got = actual[key];
  if (got !== want) {
    throw new Error(`verify failed: ${key} expected ${want}, got ${got}`);
  }
}

print("verify complete: all counts match expected seed data");
